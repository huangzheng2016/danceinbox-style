# Frontend Embed Strategy Reference

How to embed `frontend/dist` into the Go binary. Two modes: **plain** (default) and **compressed** (opt-in for large bundles). Read this when `du -sh frontend/dist/` crosses ~50 MB or the binary feels unmanageable.

## Decision Table

| Indicator | Mode | Why |
|----------|------|-----|
| Unpacked `frontend/dist/` < 50 MB | Plain | `//go:embed` is the simplest path: zero runtime steps, no extra dep, no cache dir. Linker is happy under this size. |
| Unpacked `frontend/dist/` 50-200 MB | Compressed | The Go linker slows linearly with embed size. Compressing the dist to zstd shrinks the binary ~5-10x and keeps link time under a second. Runtime extracts to user cache dir on first launch. |
| Unpacked `frontend/dist/` > 200 MB | Compressed + audit | Same mechanism, but ask first: do you really need to embed it? Maybe ship dist as a sibling tarball and read at runtime, or push assets to a CDN. Reach for the user before just enabling compression. |
| Binary distributed offline to bandwidth-constrained users | Compressed | Smaller download. |
| Binary serves only behind a CDN you control | Plain | Compression saves nothing the CDN doesn't already do; runtime extraction adds a failure mode. |

Default is **Plain**. Only switch when you have a measured reason.

## Mode A — Plain (default)

`frontend_embed.go`:

```go
package main

import (
    "embed"
    "io/fs"
    "net/http"
)

//go:embed all:frontend/dist
var frontendDist embed.FS

func getFrontendHTTPFS() http.FileSystem {
    sub, err := fs.Sub(frontendDist, "frontend/dist")
    if err != nil {
        panic(err)
    }
    return http.FS(sub)
}
```

`build.sh` needs nothing extra. Frontend builds to `frontend/dist/`, Go build sees it via `//go:embed`.

The `serveStaticFile` / `serveSPAFile` helpers that `router.go` calls are NOT repeated here — they are defined once in `backend-skeleton.md` (`frontend_embed.go` section) and live in the same `package main`. Under Mode A they stay inside `frontend_embed.go`; the Mode B migration first moves them into a separate `frontend_helpers.go` (still `package main`, step and rationale in the Migration Checklist). They take any `http.FileSystem`, so they work unchanged against Mode B's `http.Dir`-backed FS below.

No dependency on `github.com/klauspost/compress/zstd`. No `cmd/embedcompress`. No runtime cache directory.

## Mode B — Compressed (opt-in)

Pipeline:
```
frontend/dist/  --(tar)-->  frontend_dist.tar  --(zstd)-->  frontend/dist.tar.zst
                                                            |
                                          sha256 -> frontend/dist.tar.zst.sha256
                                                            |
                                                      //go:embed
                                                            |
                                                      Go binary
                                                            |
                                                  runtime: extract on first launch
                                                            |
                                            user cache dir / <module>/frontend_dist/
                                            (re-uses if SHA marker matches)
```

### Step 1: Add the dependency

```bash
go get github.com/klauspost/compress/zstd
```

### Step 2: Create `cmd/embedcompress/main.go`

```go
package main

import (
    "flag"
    "fmt"
    "io"
    "os"

    "github.com/klauspost/compress/zstd"
)

func main() {
    input := flag.String("input", "", "tar input file")
    output := flag.String("output", "", "zstd output file")
    flag.Parse()
    if *input == "" || *output == "" {
        fmt.Fprintln(os.Stderr, "usage: embedcompress -input <file> -output <file>")
        os.Exit(2)
    }
    in, err := os.Open(*input)
    if err != nil { exit("open input", err) }
    defer in.Close()
    out, err := os.Create(*output)
    if err != nil { exit("create output", err) }
    defer out.Close()

    zw, err := zstd.NewWriter(out, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
    if err != nil { exit("zstd writer", err) }
    if _, err := io.Copy(zw, in); err != nil { _ = zw.Close(); exit("compress", err) }
    if err := zw.Close(); err != nil { exit("close zstd", err) }
}

func exit(prefix string, err error) { fmt.Fprintf(os.Stderr, "%s: %v\n", prefix, err); os.Exit(1) }
```

### Step 3: Replace `frontend_embed.go`

```go
package main

import (
    "archive/tar"
    "bytes"
    "crypto/sha256"
    _ "embed"
    "encoding/hex"
    "errors"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "runtime"
    "strings"

    "github.com/klauspost/compress/zstd"
)

//go:embed frontend/dist.tar.zst
var frontendDistArchive []byte

//go:embed frontend/dist.tar.zst.sha256
var frontendDistArchiveSHA256 []byte

const frontendDistMarkerName = ".<MODULE_NAME>-frontend-dist.sha256"

func getFrontendHTTPFS() http.FileSystem {
    // Actual dist layout: index.html + assets/ at the root, admin bundle
    // under admin/ (see frontend/skeleton.md "Build Output Layout").
    // Need a separate admin FS for a dual-entry setup? Add a small fallback
    // FS yourself (primary <root>/admin, shared <root>) — nothing in the
    // default router calls one.
    return http.Dir(mustPrepareFrontendDist())
}

func mustPrepareFrontendDist() string {
    root, err := prepareFrontendDist()
    if err != nil {
        panic(err)
    }
    return root
}

func prepareFrontendDist() (string, error) {
    sha, err := frontendArchiveSHA256()
    if err != nil {
        return "", err
    }
    base, err := frontendDistBaseDir()
    if err != nil {
        return "", err
    }
    root := filepath.Join(base, "frontend_dist")
    marker := filepath.Join(root, frontendDistMarkerName)
    if data, err := os.ReadFile(marker); err == nil && strings.TrimSpace(string(data)) == sha {
        return root, nil
    }
    if err := os.RemoveAll(root); err != nil {
        return "", err
    }
    if err := os.MkdirAll(root, 0o755); err != nil {
        return "", err
    }
    if err := extractFrontendTarZstd(frontendDistArchive, root); err != nil {
        _ = os.RemoveAll(root)
        return "", err
    }
    if err := os.WriteFile(marker, []byte(sha+"\n"), 0o644); err != nil {
        _ = os.RemoveAll(root)
        return "", err
    }
    return root, nil
}

func frontendArchiveSHA256() (string, error) {
    expected := strings.TrimSpace(string(frontendDistArchiveSHA256))
    if expected == "" {
        return "", errors.New("frontend dist archive sha256 missing")
    }
    expected = strings.Fields(expected)[0]
    if len(expected) != sha256.Size*2 {
        return "", fmt.Errorf("invalid sha256: %s", expected)
    }
    if _, err := hex.DecodeString(expected); err != nil {
        return "", fmt.Errorf("invalid sha256: %w", err)
    }
    sum := sha256.Sum256(frontendDistArchive)
    actual := hex.EncodeToString(sum[:])
    if !strings.EqualFold(actual, expected) {
        return "", fmt.Errorf("sha256 mismatch: expected %s got %s", strings.ToLower(expected), actual)
    }
    return strings.ToLower(expected), nil
}

func frontendDistBaseDir() (string, error) {
    if dir, err := os.UserCacheDir(); err == nil && strings.TrimSpace(dir) != "" {
        return filepath.Join(dir, "<MODULE_NAME>"), nil
    }
    return filepath.Join(os.TempDir(), "<MODULE_NAME>"), nil
}

func extractFrontendTarZstd(payload []byte, dest string) error {
    zr, err := zstd.NewReader(bytes.NewReader(payload))
    if err != nil {
        return err
    }
    defer zr.Close()
    tr := tar.NewReader(zr)
    cleanDest, err := filepath.Abs(dest)
    if err != nil {
        return err
    }
    for {
        hdr, err := tr.Next()
        if errors.Is(err, io.EOF) {
            return nil
        }
        if err != nil {
            return err
        }
        if hdr.Name == "" {
            continue
        }
        target, err := safeArchivePath(cleanDest, hdr.Name)
        if err != nil {
            return err
        }
        switch hdr.Typeflag {
        case tar.TypeDir:
            if err := os.MkdirAll(target, os.FileMode(hdr.Mode)&0o777); err != nil {
                return err
            }
        case tar.TypeReg, tar.TypeRegA:
            if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
                return err
            }
            f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)&0o777)
            if err != nil {
                return err
            }
            _, copyErr := io.Copy(f, tr)
            closeErr := f.Close()
            if copyErr != nil {
                return copyErr
            }
            if closeErr != nil {
                return closeErr
            }
        }
    }
}

func safeArchivePath(cleanDest, name string) (string, error) {
    name = filepath.ToSlash(strings.TrimPrefix(name, "/"))
    if name == "." || name == "" {
        return cleanDest, nil
    }
    if strings.HasPrefix(name, "../") || strings.Contains(name, "/../") || strings.Contains(name, "\x00") {
        return "", fmt.Errorf("unsafe archive path: %s", name)
    }
    target := filepath.Clean(filepath.Join(cleanDest, filepath.FromSlash(name)))
    rel, err := filepath.Rel(cleanDest, target)
    if err != nil {
        return "", err
    }
    if rel == "." {
        return target, nil
    }
    if strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
        return "", fmt.Errorf("archive path escapes root: %s", name)
    }
    if runtime.GOOS == "windows" && strings.ContainsAny(rel, `:*?"<>|`) {
        return "", fmt.Errorf("unsafe archive path: %s", name)
    }
    return target, nil
}
```

### Step 4: Add to `build.sh`

Insert this helper near the top of `build.sh` (after the version helpers, before the build steps):

```bash
sha256_file() {
    if command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$1" | awk '{print $1}'
    else
        sha256sum "$1" | awk '{print $1}'
    fi
}

prepare_frontend_archive() {
    [ -d "frontend/dist" ] || { echo "frontend/dist 不存在"; exit 1; }
    mkdir -p .cache
    local tmp_tar=".cache/frontend_dist.tar"
    rm -f "$tmp_tar" "frontend/dist.tar.zst" "frontend/dist.tar.zst.sha256"
    (cd frontend/dist && COPYFILE_DISABLE=1 tar cf "../../$tmp_tar" .)
    go run ./cmd/embedcompress -input "$tmp_tar" -output "frontend/dist.tar.zst"
    sha256_file "frontend/dist.tar.zst" > "frontend/dist.tar.zst.sha256"
    rm -f "$tmp_tar"
}
```

Then uncomment the hook block in the build pipeline:

```bash
if [ "$FRONTEND_REBUILT" = true ] || [ ! -f "frontend/dist.tar.zst" ]; then
    prepare_frontend_archive
fi
```

And include the archive's SHA in the Go hash so Go rebuilds when the archive changed — append exactly one line to `calc_go_hash`'s subshell, after the `frontend=...` echo:

```bash
        echo "frontend_archive=$(sha256_file frontend/dist.tar.zst 2>/dev/null || echo missing)"
```

### Step 5: Update `.gitignore`

```
frontend/dist.tar.zst
frontend/dist.tar.zst.sha256
```

These are build artifacts. Do NOT commit them.

### Step 6: Update Dockerfile

Insert the compression step between frontend build and Go build:

```dockerfile
# After Stage 1 (frontend-builder) and inside Stage 2 (go-builder):
COPY --from=frontend-builder /src/frontend/dist ./frontend/dist
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    mkdir -p .cache && \
    tar -C frontend/dist -cf .cache/frontend_dist.tar . && \
    go run ./cmd/embedcompress -input .cache/frontend_dist.tar -output frontend/dist.tar.zst && \
    sha256sum frontend/dist.tar.zst | awk '{print $1}' > frontend/dist.tar.zst.sha256 && \
    rm -f .cache/frontend_dist.tar
# then the existing go build line.
```

## Trade-offs

**Plain (Mode A) gives up:**
- Binary size. A 100 MB frontend embeds as 100 MB extra in the binary.

**Compressed (Mode B) gives up:**
- Simplicity. You now have `cmd/embedcompress`, an extra build step, a runtime cache dir.
- Cold-start cost. First boot extracts the tar.zst to disk; subsequent boots are O(read marker) only.
- A failure mode. Marker/SHA drift re-extracts on the next boot, but files deleted under a still-matching marker are NOT detected (clear the cache dir), and an unwritable cache dir panics at boot.
- Cross-platform polish: on Windows you handle reserved path chars (`safeArchivePath` does this).

## Migration Checklist

Going from Plain to Compressed:
- [ ] `go get github.com/klauspost/compress/zstd`
- [ ] Add `cmd/embedcompress/main.go`.
- [ ] Move `serveStaticFile` / `serveSPAFile` out of `frontend_embed.go` into a new `frontend_helpers.go` (same `package main`) — they are defined only in the Mode A `frontend_embed.go`, so replacing that file without extracting them first deletes helpers `router.go` still calls.
- [ ] Replace `frontend_embed.go` with the Mode B template.
- [ ] Update `build.sh`: `prepare_frontend_archive` + GO_HASH composition.
- [ ] Update `Dockerfile`: compression step.
- [ ] Add the artifacts to `.gitignore`.
- [ ] Run `./build.sh` clean — verify the binary boots, `/ping` answers, frontend renders.
- [ ] Commit `chore: switch to compressed frontend embed (dist NN MB)`.

Going back from Compressed to Plain:
- [ ] Restore Mode A `frontend_embed.go` (either fold `serveStaticFile` / `serveSPAFile` back into it and delete `frontend_helpers.go`, or keep `frontend_helpers.go` and restore only the embed directive + `getFrontendHTTPFS` — never both, or the helpers are defined twice).
- [ ] Remove `cmd/embedcompress/` directory.
- [ ] Remove the compression step from `build.sh` and `Dockerfile`.
- [ ] Remove the artifact entries from `.gitignore`.
- [ ] `go mod tidy` to drop the zstd dep if no other code uses it.
- [ ] Delete leftover `frontend/dist.tar.zst*` files.

## Pitfalls

- **`os.UserCacheDir()` fails on some Linux containers** when `HOME` is unset and `XDG_CACHE_HOME` is unset. The helper falls back to `os.TempDir()`. Fine for stateless containers; if you want a stable cache, set `HOME=/app` in the Dockerfile.
- **First boot on a slow disk takes seconds** during extraction. If startup latency matters (e.g. you publish boot-time SLO), this is a regression vs. plain embed.
- **Multi-instance contention**: if two processes start simultaneously on a shared cache dir, they race on extraction. The marker check makes the race idempotent (last writer wins, same SHA), but you may briefly see duplicate I/O. Acceptable for single-machine deploys; not acceptable for shared NFS — use a per-instance cache dir.
- **CI binary verification**: when a CI job builds the binary and then runs `go test`, the test binary may try to extract to the CI cache dir. If your tests bypass `getFrontendHTTPFS`, no problem. If they hit it, set `HOME=$RUNNER_TEMP` so extraction lands in a clean place.
- **Build error `embed: pattern frontend/dist.tar.zst: no matching files found`**: you forgot to run `prepare_frontend_archive` (or didn't uncomment the hook in build.sh).
