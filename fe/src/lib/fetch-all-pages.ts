type Page<T> = { data: T[]; total: number };

const PAGE_SIZE = 500;

export async function fetchAllPages<T extends { id: string }>(
  fetchPage: (page: number, limit: number) => Promise<Page<T>>,
): Promise<T[]> {
  const first = await fetchPage(1, PAGE_SIZE);
  if (!Array.isArray(first.data) || !Number.isSafeInteger(first.total) || first.total < 0) {
    throw new Error("Respons pagination tidak valid");
  }

  const items = [...first.data];
  const pageCount = Math.ceil(first.total / PAGE_SIZE);
  for (let page = 2; page <= pageCount; page++) {
    const result = await fetchPage(page, PAGE_SIZE);
    if (!Array.isArray(result.data) || result.total !== first.total) {
      throw new Error("Data berubah saat ekspor; silakan coba lagi");
    }
    items.push(...result.data);
  }

  if (items.length !== first.total || new Set(items.map((item) => item.id)).size !== items.length) {
    throw new Error("Data ekspor tidak lengkap; silakan coba lagi");
  }
  return items;
}
