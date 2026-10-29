export function getOriginalExtension(encryptedFilename: string): string {
  if (!encryptedFilename) return "";
  const filename = encryptedFilename.split("/").pop() || "";
  const base = filename.replace(/\.enc$/, "");
  const current = base.match(/^\d+-[a-f0-9]+-(.+)$/);
  if (current) return current[1].toLowerCase();
  const legacy = base.match(/^\d+-[a-f0-9]+\.(.+)$/);
  return legacy?.[1]?.toLowerCase() || "bin";
}

export function isPdf(filename: string | null | undefined): boolean {
  return Boolean(filename && getOriginalExtension(filename) === "pdf");
}

export function isImage(filename: string | null | undefined): boolean {
  return Boolean(filename && ["jpg", "jpeg", "png", "webp"].includes(getOriginalExtension(filename)));
}
