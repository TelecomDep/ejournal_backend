export async function sha256Hex(text) {
  const source = new TextEncoder().encode(text);
  const digest = await crypto.subtle.digest('SHA-256', source);
  const bytes = Array.from(new Uint8Array(digest));
  return bytes.map((byte) => byte.toString(16).padStart(2, '0')).join('');
}
