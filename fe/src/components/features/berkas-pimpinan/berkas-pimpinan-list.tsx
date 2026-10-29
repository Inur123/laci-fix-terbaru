"use client";

import React, { useState, useEffect, useRef } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import Link from "next/link";
import {
  deleteBerkasPimpinan,
  getBerkasPimpinans,
  getBerkasPimpinanDownloadToken,
} from "@/app/actions/berkas-pimpinan-actions";
import { logExport } from "@/app/actions/log-activity-actions";
import { ConfirmModal } from "@/components/shared/confirm-modal";
import { toast } from "sonner";
import { Label } from "@/components/ui/label";
import XLSX from "@/lib/excel";
import { fetchAllPages } from "@/lib/fetch-all-pages";
import { cn } from "@/lib/utils";
import {
  Eye,
  Pencil,
  Trash2,
  Search,
  FileSpreadsheet,
  ArrowUpDown,
  ArrowUp,
  ArrowDown,
} from "lucide-react";
import { Spinner } from "@/components/ui/spinner";

type BerkasPimpinan = {
  id: string;
  nama: string;
  tanggal: Date;
  catatan: string | null;
  file: string | null;
};

export function BerkasPimpinanList({
  berkasList: initialBerkasList,
  userRole,
  totalPages: initialTotalPages,
  currentPage: initialCurrentPage,
  totalItems: initialTotalItems,
}: {
  berkasList: BerkasPimpinan[];
  userRole: string;
  totalPages: number;
  currentPage: number;
  totalItems: number;
}) {
  const [data, setData] = useState<BerkasPimpinan[]>(initialBerkasList);
  const [totalPages, setTotalPages] = useState(initialTotalPages);
  const [currentPage, setCurrentPage] = useState(initialCurrentPage);
  const [totalItems, setTotalItems] = useState(initialTotalItems);
  const [isLoading, setIsLoading] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [loadError, setLoadError] = useState("");
  const [searchTerm, setSearchTerm] = useState("");

  type SortKey = "nama" | "tanggal" | "catatan";
  type SortDir = "asc" | "desc";
  const [sortKey, setSortKey] = useState<SortKey | null>(null);
  const [sortDir, setSortDir] = useState<SortDir>("desc");

  const isFirstRender = useRef(true);

  const fetchData = async (
    query: string,
    page: number,
    sKey: SortKey | null = sortKey,
    sDir: SortDir = sortDir,
  ) => {
    setIsLoading(true);
    setLoadError("");
    try {
      const result = await getBerkasPimpinans(query, page, 10, sKey, sDir);
      setData(result.data as BerkasPimpinan[]);
      setTotalPages(result.totalPages);
      setTotalItems(result.total);
    } catch (error) {
      console.error("Error fetching data:", error);
      const errMsg = error instanceof Error ? error.message : String(error);
      if (
        !errMsg.includes("unexpected response") &&
        !errMsg.includes("NEXT_REDIRECT") &&
        !errMsg.includes("abort")
      ) {
        setLoadError("Data berkas gagal dimuat. Silakan coba lagi.");
        toast.error(`Gagal memuat data: ${errMsg}`);
      }
    } finally {
      setIsLoading(false);
    }
  };

  const handleSort = (key: SortKey) => {
    let newDir: SortDir = "asc";
    if (sortKey === key) {
      newDir = sortDir === "asc" ? "desc" : "asc";
      setSortDir(newDir);
    } else {
      setSortKey(key);
      setSortDir("asc");
    }
    setCurrentPage(1);
    fetchData(searchTerm, 1, key, newDir);
  };

  const SortIcon = ({ col }: { col: SortKey }) => {
    if (sortKey !== col)
      return (
        <ArrowUpDown className="ml-1.5 h-3.5 w-3.5 text-slate-400 inline-block" />
      );
    return sortDir === "asc" ? (
      <ArrowUp className="ml-1.5 h-3.5 w-3.5 text-slate-600 inline-block" />
    ) : (
      <ArrowDown className="ml-1.5 h-3.5 w-3.5 text-slate-600 inline-block" />
    );
  };

  const sortState = (column: SortKey) =>
    sortKey === column
      ? sortDir === "asc"
        ? "ascending"
        : "descending"
      : "none";

  const sortedData = data;

  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);
  const realtimeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const deleteTargetName = data.find(
    (item) => item.id === confirmDeleteId,
  )?.nama;

  useEffect(() => {
    if (isFirstRender.current) {
      isFirstRender.current = false;
      return;
    }
    const timer = setTimeout(() => {
      setCurrentPage(1);
      fetchData(searchTerm, 1);
    }, 500);
    return () => clearTimeout(timer);
  }, [searchTerm]);

  useEffect(() => {
    const handler = (event: Event) => {
      const detail = (event as CustomEvent).detail as {
        type?: string;
        model?: string;
      };
      if (!detail || detail.type !== "mutation") return;
      if (detail.model !== "BerkasPimpinan") return;
      if (realtimeTimerRef.current) return;
      realtimeTimerRef.current = setTimeout(() => {
        realtimeTimerRef.current = null;
        fetchData(searchTerm, currentPage);
      }, 300);
    };
    window.addEventListener("laci-realtime", handler as EventListener);
    return () => {
      window.removeEventListener("laci-realtime", handler as EventListener);
      if (realtimeTimerRef.current) {
        clearTimeout(realtimeTimerRef.current);
        realtimeTimerRef.current = null;
      }
    };
  }, [searchTerm, currentPage]);

  const handlePageChange = (page: number) => {
    setCurrentPage(page);
    fetchData(searchTerm, page);
  };

  const handleDelete = async () => {
    if (!confirmDeleteId || isDeleting) return;
    const id = confirmDeleteId;
    setIsDeleting(true);
    try {
      const result = await deleteBerkasPimpinan(id);
      if (result.error) {
        toast.error(result.error);
        return;
      }

      setConfirmDeleteId(null);
      toast.success("Berkas berhasil dihapus");
      await fetchData(searchTerm, currentPage);
    } finally {
      setIsDeleting(false);
    }
  };

  const handlePreview = async (id: string) => {
    const previewWindow = window.open("about:blank", "_blank");
    if (previewWindow) previewWindow.opener = null;

    try {
      const token = await getBerkasPimpinanDownloadToken(id);
      const url = `${window.location.origin}/api/berkas-pimpinan/download/${encodeURIComponent(id)}?preview=true&token=${encodeURIComponent(token)}`;
      if (previewWindow) {
        previewWindow.location.replace(url);
      } else {
        window.location.href = url;
      }
    } catch {
      previewWindow?.close();
      toast.error("Gagal membuka file berkas");
    }
  };

  const handleExportExcel = async () => {
    if (totalItems === 0) {
      toast.error("Tidak ada data untuk diexport");
      return;
    }

    toast.info("Menyiapkan data export...");

    const dateStr = new Date().toLocaleDateString("id-ID").replace(/\//g, "-");
    const moduleName =
      userRole === "SEKRETARIS_CABANG" ? "Berkas_Cabang" : "Berkas_PAC";
    const filename = `${moduleName}_${dateStr}.xlsx`;

    let allData = data;
    if (totalItems > data.length) {
      try {
        allData = await fetchAllPages<BerkasPimpinan>((page, limit) =>
          getBerkasPimpinans(searchTerm, page, limit, sortKey, sortDir),
        );
      } catch {
        toast.error("Gagal mengambil semua data untuk export");
        return;
      }
    }

    const exportData: Record<string, string | number>[] = allData.map(
      (item, index) => ({
        No: index + 1,
        Nama: item.nama,
        Tanggal: new Date(item.tanggal).toLocaleDateString("id-ID", {
          day: "numeric",
          month: "long",
          year: "numeric",
        }),
        Catatan: item.catatan || "-",
      }),
    );

    const worksheet = XLSX.utils.json_to_sheet(exportData);
    const range = XLSX.utils.decode_range(worksheet["!ref"] || "A1:D1");
    const headerColor = userRole === "SEKRETARIS_CABANG" ? "3b82f6" : "10b981";

    const headerStyle = {
      font: { name: "Arial", bold: true, color: { rgb: "FFFFFF" } },
      fill: { fgColor: { rgb: headerColor } },
      alignment: { horizontal: "center", vertical: "center" },
      border: {
        top: { style: "thin", color: { rgb: headerColor } },
        bottom: { style: "thin", color: { rgb: headerColor } },
        left: { style: "thin", color: { rgb: headerColor } },
        right: { style: "thin", color: { rgb: headerColor } },
      },
    };

    for (let C = range.s.c; C <= range.e.c; ++C) {
      const address = XLSX.utils.encode_cell({ r: 0, c: C });
      if (worksheet[address]) worksheet[address].s = headerStyle;
    }

    const wscols = Object.keys(exportData[0] || {}).map((key) => {
      const maxLen = Math.max(
        key.length,
        ...exportData.map((row) => String(row[key] || "").length),
      );
      return { wch: Math.min(maxLen + 2, 50) };
    });
    worksheet["!cols"] = wscols;

    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, worksheet, "Berkas");
    await XLSX.writeFile(workbook, filename);
    logExport("BERKAS_PIMPINAN", filename);
    toast.success("File excel berhasil didownload!");
  };

  const listLabel =
    userRole === "SEKRETARIS_CABANG" ? "Berkas Cabang" : "Berkas PAC";
  const displayedDeleteName = deleteTargetName
    ? deleteTargetName.length > 60
      ? `${deleteTargetName.slice(0, 57)}...`
      : deleteTargetName
    : "ini";

  return (
    <div className="flex flex-col">
      <div className="flex flex-col md:flex-row gap-4 mb-4 items-end">
        <div className="flex-1 relative w-full">
          <Label
            htmlFor="berkas-pimpinan-search"
            className="text-xs font-medium mb-1 block"
          >
            Cari Berkas
          </Label>
          <div className="relative">
            <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              id="berkas-pimpinan-search"
              placeholder="Cari nama atau catatan..."
              className="pl-9 w-full bg-white h-9 text-sm"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
            />
          </div>
        </div>

        <div className="w-full md:w-auto">
          <Button
            variant="outline"
            className="h-9 w-full md:w-auto px-4 text-sm bg-white border-slate-200 shadow-sm whitespace-nowrap"
            onClick={handleExportExcel}
            disabled={isLoading}
            aria-label={`Export ${listLabel} ke Excel`}
          >
            <FileSpreadsheet
              className={cn(
                "mr-2 h-3.5 w-3.5",
                userRole === "SEKRETARIS_CABANG"
                  ? "text-blue-600"
                  : "text-green-600",
              )}
            />
            Export
          </Button>
        </div>
      </div>

      <div className="relative">
        <div className="rounded-md border">
          <div className="overflow-x-auto">
            {isLoading && (
              <div className="absolute inset-0 z-20 flex items-center justify-center gap-2 rounded-md bg-white/75 text-sm text-slate-700">
                <Spinner className="h-5 w-5" />
                Memuat data...
              </div>
            )}
            <Table
              className="w-full min-w-[850px] table-fixed [&_td]:py-2 [&_th]:py-2"
              aria-busy={isLoading}
            >
              <TableHeader className="sticky top-0 bg-white z-10">
                <TableRow>
                  <TableHead className="w-[50px] text-center bg-slate-50/40 whitespace-nowrap">
                    No
                  </TableHead>
                  <TableHead
                    className="w-[220px] bg-slate-50/40 whitespace-nowrap"
                    aria-sort={sortState("nama")}
                  >
                    <button
                      type="button"
                      className="inline-flex min-h-11 w-full items-center rounded-sm text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      onClick={() => handleSort("nama")}
                    >
                      Nama
                      <SortIcon col="nama" />
                    </button>
                  </TableHead>
                  <TableHead
                    className="w-[150px] bg-slate-50/40 whitespace-nowrap"
                    aria-sort={sortState("tanggal")}
                  >
                    <button
                      type="button"
                      className="inline-flex min-h-11 w-full items-center rounded-sm text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      onClick={() => handleSort("tanggal")}
                    >
                      Tanggal
                      <SortIcon col="tanggal" />
                    </button>
                  </TableHead>
                  <TableHead
                    className="w-[300px] bg-slate-50/40 whitespace-nowrap"
                    aria-sort={sortState("catatan")}
                  >
                    <button
                      type="button"
                      className="inline-flex min-h-11 w-full items-center rounded-sm text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      onClick={() => handleSort("catatan")}
                    >
                      Catatan
                      <SortIcon col="catatan" />
                    </button>
                  </TableHead>
                  <TableHead className="w-[130px] text-right bg-slate-50/40 whitespace-nowrap">
                    Aksi
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loadError ? (
                  <TableRow>
                    <TableCell
                      colSpan={5}
                      className="h-32 text-center text-red-600"
                    >
                      {loadError}
                    </TableCell>
                  </TableRow>
                ) : sortedData.length === 0 ? (
                  <TableRow>
                    <TableCell
                      colSpan={5}
                      className="h-32 text-center text-muted-foreground"
                    >
                      {searchTerm
                        ? "Tidak ada data berkas pimpinan yang cocok dengan filter."
                        : "Belum ada data berkas pimpinan."}
                    </TableCell>
                  </TableRow>
                ) : (
                  sortedData.map((item, index) => (
                    <TableRow key={item.id}>
                      <TableCell className="text-center text-muted-foreground font-medium whitespace-nowrap">
                        {(currentPage - 1) * 10 + index + 1}
                      </TableCell>
                      <TableCell
                        className="font-medium whitespace-nowrap"
                        title={item.nama}
                      >
                        <div className="truncate">{item.nama}</div>
                      </TableCell>
                      <TableCell className="whitespace-nowrap pr-6">
                        {new Date(item.tanggal).toLocaleDateString("id-ID", {
                          day: "2-digit",
                          month: "short",
                          year: "numeric",
                        })}
                      </TableCell>
                      <TableCell className="whitespace-nowrap">
                        <div className="truncate" title={item.catatan || ""}>
                          {item.catatan || "-"}
                        </div>
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-2">
                          {item.file && (
                            <Button
                              size="sm"
                              variant="outline"
                              className="h-11 w-11 p-0"
                              title={`Tampilkan file ${item.nama}`}
                              aria-label={`Tampilkan file Berkas ${item.nama}`}
                              onClick={() => handlePreview(item.id)}
                            >
                              <Eye className="w-4 h-4" />
                            </Button>
                          )}
                          <Button
                            size="sm"
                            variant="outline"
                            className="h-11 w-11 p-0"
                            asChild
                            title={`Edit ${item.nama}`}
                          >
                            <Link
                              href={`/dashboard/berkas-pimpinan/${item.id}/edit`}
                              aria-label={`Edit Berkas ${item.nama}`}
                            >
                              <Pencil className="w-4 h-4" />
                            </Link>
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            className="h-11 w-11 p-0 text-red-500 hover:text-red-600"
                            onClick={() => setConfirmDeleteId(item.id)}
                            title={`Hapus ${item.nama}`}
                            aria-label={`Hapus Berkas ${item.nama}`}
                          >
                            <Trash2 className="w-4 h-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))
                )}

                {totalPages >= 1 && (
                  <TableRow className="hover:bg-transparent border-t bg-white">
                    <TableCell colSpan={5} className="p-0">
                      <div className="flex items-center justify-start sm:justify-between px-4 py-2">
                        <p className="text-xs text-muted-foreground hidden sm:block">
                          Menampilkan{" "}
                          <span className="font-medium text-slate-700">
                            {(currentPage - 1) * 10 + 1}
                          </span>{" "}
                          sampai{" "}
                          <span className="font-medium text-slate-700">
                            {Math.min(currentPage * 10, totalItems)}
                          </span>{" "}
                          dari{" "}
                          <span className="font-medium text-slate-700">
                            {totalItems}
                          </span>{" "}
                          berkas
                        </p>
                        <Pagination className="mx-0 w-auto scale-90 sm:scale-100 origin-left">
                          <PaginationContent>
                            <PaginationItem>
                              <PaginationPrevious
                                href="#"
                                aria-disabled={currentPage === 1}
                                tabIndex={currentPage === 1 ? -1 : 0}
                                onClick={(e) => {
                                  e.preventDefault();
                                  if (currentPage > 1)
                                    handlePageChange(currentPage - 1);
                                }}
                                className={
                                  currentPage === 1
                                    ? "pointer-events-none opacity-50"
                                    : "cursor-pointer"
                                }
                              />
                            </PaginationItem>

                            {[...Array(totalPages)].map((_, i) => {
                              const page = i + 1;
                              if (
                                page === 1 ||
                                page === totalPages ||
                                (page >= currentPage - 1 &&
                                  page <= currentPage + 1)
                              ) {
                                return (
                                  <PaginationItem key={page}>
                                    <PaginationLink
                                      href="#"
                                      onClick={(e) => {
                                        e.preventDefault();
                                        handlePageChange(page);
                                      }}
                                      isActive={currentPage === page}
                                      className="cursor-pointer"
                                    >
                                      {page}
                                    </PaginationLink>
                                  </PaginationItem>
                                );
                              } else if (
                                page === currentPage - 2 ||
                                page === currentPage + 2
                              ) {
                                return <PaginationEllipsis key={page} />;
                              }
                              return null;
                            })}

                            <PaginationItem>
                              <PaginationNext
                                href="#"
                                aria-disabled={currentPage === totalPages}
                                tabIndex={currentPage === totalPages ? -1 : 0}
                                onClick={(e) => {
                                  e.preventDefault();
                                  if (currentPage < totalPages)
                                    handlePageChange(currentPage + 1);
                                }}
                                className={
                                  currentPage === totalPages
                                    ? "pointer-events-none opacity-50"
                                    : "cursor-pointer"
                                }
                              />
                            </PaginationItem>
                          </PaginationContent>
                        </Pagination>
                      </div>
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </div>
      </div>

      <ConfirmModal
        isOpen={!!confirmDeleteId}
        onClose={() => {
          if (!isDeleting) setConfirmDeleteId(null);
        }}
        onConfirm={handleDelete}
        title="Hapus Berkas?"
        description={`Apakah Anda yakin ingin menghapus berkas "${displayedDeleteName}"?`}
        variant="destructive"
        loading={isDeleting}
      />
    </div>
  );
}
