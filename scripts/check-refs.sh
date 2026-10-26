#!/usr/bin/env bash
set -u
cd "$(dirname "$0")/.."

fail=0

echo "== check 0: frontmatter parses =="
command -v python3 >/dev/null && python3 - <<'EOF' || echo "python3 unavailable, skipped"
import sys
try:
    import yaml
except ImportError:
    print("pyyaml unavailable, skipped"); sys.exit(0)
fm = open('SKILL.md').read().split('---')[1]
d = yaml.safe_load(fm)
assert isinstance(d, dict) and d.get('name') and d.get('description'), "frontmatter missing name/description"
print("ok")
EOF
[ $? -ne 0 ] && fail=1

echo "== check 1: referenced paths exist =="
grep -rhoE '`((backend|frontend)/[A-Za-z0-9_/.-]+\.(md|css|vue|ts))`' \
    --include='*.md' SKILL.md backend frontend | tr -d '`' | sort -u | \
while read -r p; do
    [ -f "$p" ] || { echo "MISSING: $p"; exit 1; }
done || fail=1

echo "== check 2: File Index <-> disk =="
indexed=$(sed -n '/^## File Index/,/^## Quick Start/p' SKILL.md | \
    grep -oE '`(backend|frontend)/[A-Za-z0-9_/.-]+\.(md|css|vue|ts)`' | tr -d '`' | sort -u)
ondisk=$(find backend frontend -type f \( -name '*.md' -o -name '*.css' -o -name '*.vue' -o -name '*.ts' \) | sort -u)
missing_on_disk=$(comm -23 <(echo "$indexed") <(echo "$ondisk"))
not_indexed=$(comm -13 <(echo "$indexed") <(echo "$ondisk"))
[ -n "$missing_on_disk" ] && { echo "indexed but missing:"; echo "$missing_on_disk"; fail=1; }
[ -n "$not_indexed" ] && { echo "on disk but not indexed:"; echo "$not_indexed"; fail=1; }
[ -z "$missing_on_disk" ] && [ -z "$not_indexed" ] && echo "ok"

echo "== check 3: CSS classes referenced in docs are defined (warn-only) =="
defined=$(cat frontend/base.css frontend/code-editor.css frontend/tokens.css frontend/templates/*.vue | \
    grep -oE '\.[a-zA-Z][a-zA-Z0-9_-]*' | sort -u)
referenced=$(grep -rhoE '`\.[a-z][a-z0-9-]*`' --include='*.md' SKILL.md backend frontend | tr -d '`' | sort -u)
allowlist='^\.(page-card|page-header|my-input|my-card|env|gitignore|dockerignore|prettierignore|hz)$'
orphans=$(comm -23 <(echo "$referenced") <(echo "$defined") | grep -vE "$allowlist" || true)
if [ -n "$orphans" ]; then
    echo "referenced but not defined anywhere (may be intentional):"
    echo "$orphans"
else
    echo "ok"
fi

echo "== check 4: design profiles, fonts, tokens, template imports =="
python3 - <<'EOF'
from pathlib import Path
import re
import sys

errors = []

def check(condition, message):
    if not condition:
        errors.append(message)

def props(block):
    return dict(re.findall(r'^\s*(--[a-zA-Z0-9-]+):\s*([^;]+);', block, re.M))

def profile_block(source, kind, name):
    match = re.search(r":root\[data-" + kind + "='" + re.escape(name) + r"'\]\s*\{(.*?)^\}", source, re.M | re.S)
    return match.group(1) if match else None

tokens_path = Path('frontend/tokens.css')
tokens = tokens_path.read_text()
root_match = re.search(r'^:root\s*\{(.*?)^\}', tokens, re.M | re.S)
dark_match = re.search(r'^:root\.dark\s*\{(.*?)^\}', tokens, re.M | re.S)
check(root_match is not None, 'missing default :root block')
check(dark_match is not None, 'missing :root.dark block')
if not root_match or not dark_match:
    print('\n'.join(errors))
    sys.exit(1)

root_props = props(root_match.group(1))
radius_names = ['xs', 'sm', 'md', 'lg', 'xl', '2xl']
shape_expected = {
    'micro': ['2px', '4px', '4px', '6px', '8px', '10px'],
    'rounded': ['8px', '12px', '16px', '20px', '24px', '28px'],
    'square': ['0', '0', '0', '0', '0', '0'],
}
for shape, expected in shape_expected.items():
    block = root_match.group(1) if shape == 'micro' else profile_block(tokens, 'shape', shape)
    check(block is not None, f'missing shape profile: {shape}')
    if block is not None:
        values = props(block)
        actual = [values.get(f'--radius-{name}') for name in radius_names]
        check(actual == expected, f'{shape} radius mismatch: {actual}')

surface_names = [
    '--app-bg', '--surface-strong', '--surface-soft', '--surface-elevated',
    '--aside-bg', '--header-bg', '--input-bg', '--input-bg-hover',
    '--surface-sticky', '--surface-sticky-header',
]
surface_expected = {
    'soft-white': ['#f8f8f6', '#efefec', '#e6e6e2', '#f8f8f6', '#efefec', '#efefec', '#f8f8f6', '#ffffff', '#f8f8f6', '#e6e6e2'],
    'white': ['#ffffff', '#f5f5f3', '#e9e9e6', '#ffffff', '#f5f5f3', '#f5f5f3', '#ffffff', '#f5f5f3', '#ffffff', '#e9e9e6'],
    'warm-paper': ['#f5f4f0', '#ecebe6', '#e3e2dc', '#f5f4f0', '#ecebe6', '#ecebe6', '#f5f4f0', '#ffffff', '#f5f4f0', '#e3e2dc'],
}
for surface, expected in surface_expected.items():
    block = root_match.group(1) if surface == 'soft-white' else profile_block(tokens, 'surface', surface)
    check(block is not None, f'missing surface profile: {surface}')
    if block is not None:
        values = props(block)
        actual = [values.get(name) for name in surface_names]
        check(actual == expected, f'{surface} surface mismatch: {actual}')

dark_props = props(dark_match.group(1))
for name in surface_names:
    check(name in dark_props, f'dark palette missing {name}')
check(tokens.find(':root.dark') > tokens.find(":root[data-surface='white']"), 'dark palette must follow light profile overrides')

local_tokens = {'--split-master-width', '--row-overlay', '--dialog-width', '--edit-grid-label-width'}
for path in [Path('frontend/base.css'), Path('frontend/code-editor.css'), *Path('frontend/templates').glob('*.vue')]:
    source = path.read_text()
    for name in set(re.findall(r'var\((--[a-zA-Z0-9-]+)', source)):
        check(name in root_props or name in dark_props or name in local_tokens, f'{path}: undefined token {name}')
    check('--radius-pill' not in source, f'{path}: retired pill control token still referenced')

font_expectations = {
    'plus-jakarta-sans-latin.woff2': ('plus-jakarta-sans-OFL.txt', 'Plus Jakarta Sans Project Authors'),
    'geist-mono-latin.woff2': ('geist-mono-OFL.txt', 'Geist Project Authors'),
}
for font, (license_name, copyright_owner) in font_expectations.items():
    font_path = Path('frontend/assets/fonts') / font
    license_path = Path('frontend/assets/fonts') / license_name
    check(font_path.is_file() and font_path.stat().st_size > 0, f'missing or empty font: {font}')
    check(license_path.is_file(), f'missing font license: {license_name}')
    if license_path.is_file():
        license_text = license_path.read_text()
        check(copyright_owner in license_text and 'SIL OPEN FONT LICENSE' in license_text, f'invalid font license: {license_name}')

for path in [*Path('frontend/templates').glob('*.vue'), *Path('frontend/templates').glob('*.ts')]:
    source = path.read_text()
    for specifier in re.findall(r"from\s+['\"](\.[^'\"]+)['\"]", source):
        base = (path.parent / specifier).resolve()
        candidates = [base] if base.suffix else [base.with_suffix('.ts'), base.with_suffix('.vue'), base.with_suffix('.css')]
        check(any(candidate.is_file() for candidate in candidates), f'{path}: missing import {specifier}')

for path in Path('frontend').rglob('*'):
    if path.suffix in {'.md', '.css', '.vue', '.ts'}:
        check('fonts.googleapis.com' not in path.read_text(), f'{path}: external font CDN remains')

if errors:
    print('\n'.join(errors))
    sys.exit(1)
print('ok')
EOF
[ $? -ne 0 ] && fail=1

[ "$fail" -eq 0 ] && echo "PASS" || { echo "FAIL"; exit 1; }
