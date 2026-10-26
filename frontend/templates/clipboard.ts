// copy verbatim; edit only the marked regions
export async function copyText(s: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(s)
    return true
  } catch {
    return false
  }
}
