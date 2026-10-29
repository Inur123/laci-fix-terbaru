import assert from "node:assert/strict";
import { test } from "node:test";
import { fetchAllPages } from "../src/lib/fetch-all-pages.ts";

test("mengambil semua halaman saat total melampaui batas backend", async () => {
  const records = Array.from({ length: 1001 }, (_, index) => ({ id: String(index) }));
  const requested = [];
  const result = await fetchAllPages(async (page, limit) => {
    requested.push([page, limit]);
    return { data: records.slice((page - 1) * limit, page * limit), total: records.length };
  });

  assert.deepEqual(requested, [[1, 500], [2, 500], [3, 500]]);
  assert.deepEqual(result, records);
});

test("menolak ekspor yang terpotong", async () => {
  await assert.rejects(
    fetchAllPages(async (page) => ({
      data: page === 1 ? Array.from({ length: 500 }, (_, index) => ({ id: String(index) })) : [],
      total: 501,
    })),
    /tidak lengkap/,
  );
});

test("menolak duplikasi akibat perubahan data selama pagination", async () => {
  await assert.rejects(
    fetchAllPages(async (page) => ({
      data: page === 1
        ? Array.from({ length: 500 }, (_, index) => ({ id: String(index) }))
        : [{ id: "499" }],
      total: 501,
    })),
    /tidak lengkap/,
  );
});
