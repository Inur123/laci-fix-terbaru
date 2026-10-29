"use client";

import React, { useState, useEffect, useRef } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { deleteBerkasSP, getBerkasSPs } from "@/app/actions/berkas-sp-actions";
import { logExport } from "@/app/actions/log-activity-actions";
import { ConfirmModal } from "@/components/shared/confirm-modal";
import { toast } from "sonner";
import { cn } from "@/lib/utils";
import { Label } from "@/components/ui/label";
import XLSX from "@/lib/excel";
import { fetchAllPages } from "@/lib/fetch-all-pages";
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

const capitalizeName = (name: string) => {
  if (!name) return "";
  return name
    .split(" ")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
    .join(" ");
};

type BerkasSP = {
  id: string;
  nama: string;
  organisasi: string | null;
  tanggalMulai: Date;
  tanggalBerakhir: Date;
  catatan: string | null;
  file: string | null;
};

const organisasiConfig: Record<string, { label: string; className: string }> = {
  IPNU: {
    label: "IPNU",
    className:
      "bg-emerald-100/80 text-emerald-700 border-emerald-200 hover:bg-emerald-200/80",
  },
  IPPNU: {
    label: "IPPNU",
    className:
      "bg-rose-100/80 text-rose-700 border-rose-200 hover:bg-rose-200/80",
  },
};

const getStatusBadge = (tanggalBerakhir: Date) => {
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const end = new Date(tanggalBerakhir);
  end.setHours(0, 0, 0, 0);

  const diffTime = end.getTime() - today.getTime();
  const diffDays = Math.round(diffTime / (1000 * 60 * 60 * 24));

  if (diffDays < 0) {
    return (
      <Badge
        variant="outline"
        className="bg-rose-50 text-rose-700 border-rose-200 mt-1 text-[10px] leading-3 h-5 px-1.5 font-medium whitespace-nowrap"
      >
        Kedaluwarsa (Lewat {Math.abs(diffDays)} Hari)
      </Badge>
    );
  } else if (diffDays === 0) {
    return (
      <Badge
        variant="outline"
        className="bg-amber-50 text-amber-700 border-amber-200 mt-1 text-[10px] leading-3 h-5 px-1.5 font-semibold whitespace-nowrap animate-pulse"
      >
        Berakhir Hari Ini!
      </Badge>
    );
  } else if (diffDays <= 30) {
    return (
      <Badge
        variant="outline"
        className="bg-orange-50 text-orange-700 border-orange-200 mt-1 text-[10px] leading-3 h-5 px-1.5 whitespace-nowrap"
      >
        Hampir Habis (Sisa {diffDays} Hari)
      </Badge>
    );
  } else {
    return (
      <Badge
        variant="outline"
        className="bg-emerald-50 text-emerald-700 border-emerald-200 mt-1 text-[10px] leading-3 h-5 px-1.5 whitespace-nowrap"
      >
        Aktif (Sisa {diffDays} Hari)
      </Badge>
    );
  }
};

export function BerkasSPList({
  berkasSPList: initialBerkasSPList,
  userRole,
  totalPages: initialTotalPages,
  currentPage: initialCurrentPage,
  totalItems: initialTotalItems,
}: {
  berkasSPList: BerkasSP[];
  userRole: string;
  totalPages: number;
  currentPage: number;
  totalItems: number;
}) {
  // Local data state
  const [data, setData] = useState<BerkasSP[]>(initialBerkasSPList);
  const [totalPages, setTotalPages] = useState(initialTotalPages);
  const [currentPage, setCurrentPage] = useState(initialCurrentPage);
  const [totalItems, setTotalItems] = useState(initialTotalItems);
  const [isLoading, setIsLoading] = useState(false);

  // Filter state
  const [searchTerm, setSearchTerm] = useState("");
  const [orgFilter, setOrgFilter] = useState("ALL");

  // Sort state
  type SortKey =
    | "nama"
    | "tanggalMulai"
    | "tanggalBerakhir"
    | "organisasi"
    | "catatan"
    | "status";
  type SortDir = "asc" | "desc";
  const [sortKey, setSortKey] = useState<SortKey | null>(null);
  const [sortDir, setSortDir] = useState<SortDir>("desc");

  const isFirstRender = useRef(true);

  // Function to fetch data with sorting parameters
  const fetchData = async (
    query: string,
    org: string,
    page: number,
    sKey: SortKey | null = sortKey,
    sDir: SortDir = sortDir,
  ) => {
    setIsLoading(true);
    try {
      const result = await getBerkasSPs(query, org, page, 10, sKey, sDir);
      setData(result.data as BerkasSP[]);
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
    fetchData(searchTerm, orgFilter, 1, key, newDir);
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

  // Server sudah melakukan pengurutan secara global, sehingga klien cukup langsung menampilkan datanya
  const sortedData = data;

  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);
  const [optimisticHiddenIds, setOptimisticHiddenIds] = useState<string[]>([]);
  const realtimeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Debounce search
  useEffect(() => {
    if (isFirstRender.current) {
      isFirstRender.current = false;
      return;
    }
    const timer = setTimeout(() => {
      setCurrentPage(1);
      fetchData(searchTerm, orgFilter, 1);
    }, 500);
    return () => clearTimeout(timer);
  }, [searchTerm, orgFilter]);

  // Realtime listener
  useEffect(() => {
    const handler = (event: Event) => {
      const detail = (event as CustomEvent).detail as {
        type?: string;
        model?: string;
      };
      if (!detail || detail.type !== "mutation") return;
      if (detail.model !== "BerkasSP") return;
      if (realtimeTimerRef.current) return;
      realtimeTimerRef.current = setTimeout(() => {
        realtimeTimerRef.current = null;
        fetchData(searchTerm, orgFilter, currentPage);
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
  }, [searchTerm, orgFilter, currentPage]);

  const handleOrgFilterChange = (val: string) => {
    setOrgFilter(val);
    setCurrentPage(1);
  };

  const handlePageChange = (page: number) => {
    setCurrentPage(page);
    fetchData(searchTerm, orgFilter, page);
  };

  const handleDelete = async () => {
    if (!confirmDeleteId) return;

    const id = confirmDeleteId;
    setConfirmDeleteId(null);
    setOptimisticHiddenIds((prev) => [...prev, id]);

    const result = await deleteBerkasSP(id);

    if (result.error) {
      setOptimisticHiddenIds((prev) => prev.filter((pid) => pid !== id));
      toast.error(result.error);
    } else {
      toast.success("Berkas SP berhasil dihapus");
      fetchData(searchTerm, orgFilter, currentPage);
    }
  };

  const handleExportExcel = async () => {
    if (totalItems === 0) {
      toast.error("Tidak ada data untuk diexport");
      return;
    }

    toast.info("Menyiapkan data export...");

    const dateStr = new Date().toLocaleDateString("id-ID").replace(/\//g, "-");
    const filename = `Berkas_SP_${dateStr}.xlsx`;

    let allData = data;
    if (totalItems > data.length) {
      try {
        allData = await fetchAllPages<BerkasSP>((page, limit) =>
          getBerkasSPs(searchTerm, orgFilter, page, limit, sortKey, sortDir),
        );
      } catch {
        toast.error("Gagal mengambil semua data untuk export");
        return;
      }
    }

    const exportData: Record<string, string | number>[] = allData.map(
      (item, index) => ({
        No: index + 1,
        "Nama Pimpinan": item.nama,
        Organisasi: item.organisasi || "-",
        "Tanggal Mulai": new Date(item.tanggalMulai).toLocaleDateString(
          "id-ID",
          {
            day: "numeric",
            month: "long",
            year: "numeric",
          },
        ),
        "Tanggal Berakhir": new Date(item.tanggalBerakhir).toLocaleDateString(
          "id-ID",
          {
            day: "numeric",
            month: "long",
            year: "numeric",
          },
        ),
        Catatan: item.catatan || "-",
      }),
    );

    const worksheet = XLSX.utils.json_to_sheet(exportData);
    const range = XLSX.utils.decode_range(worksheet["!ref"] || "A1:G1");
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
      if (!worksheet[address]) continue;
      worksheet[address].s = headerStyle;
    }

    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, worksheet, "Berkas SP");

    const wscols = Object.keys(exportData[0] || {}).map((key) => {
      const maxLen = Math.max(
        key.length,
        ...exportData.map((row) => String(row[key] || "").length),
      );
      return { wch: Math.min(maxLen + 2, 50) };
    });
    worksheet["!cols"] = wscols;

    await XLSX.writeFile(workbook, filename);
    logExport("BERKAS_SP", filename);
    toast.success("File excel berhasil didownload!");
  };

  return (
    <div className="flex flex-col">
      <div className="flex flex-col md:flex-row gap-4 mb-4 items-end">
        <div className="flex-2 relative w-full">
          <Label
            htmlFor="berkas-sp-search"
            className="text-xs font-medium mb-1 block"
          >
            Cari Berkas
          </Label>
          <div className="relative">
            <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              id="berkas-sp-search"
              placeholder="Cari nama pimpinan atau catatan..."
              className="pl-9 w-full bg-white h-9 text-sm"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
            />
          </div>
        </div>

        <div className="w-full md:w-72">
          <Label
            htmlFor="berkas-sp-organisasi-filter"
            className="text-xs font-medium mb-1 block"
          >
            Organisasi
          </Label>
          <Select value={orgFilter} onValueChange={handleOrgFilterChange}>
            <SelectTrigger
              id="berkas-sp-organisasi-filter"
              aria-label="Filter organisasi Berkas SP"
              className="w-full bg-white h-9 text-sm"
            >
              <SelectValue placeholder="Semua" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="ALL">Semua</SelectItem>
              <SelectItem value="IPNU">IPNU</SelectItem>
              <SelectItem value="IPPNU">IPPNU</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="w-full md:w-auto">
          <Button
            variant="outline"
            className="h-9 w-full md:w-auto px-4 text-sm bg-white border-slate-200 shadow-sm whitespace-nowrap"
            onClick={handleExportExcel}
            disabled={isLoading}
            aria-label="Export Berkas SP ke Excel"
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

      {/* Table Section */}
      <div className="relative">
        <div className="rounded-md border">
          <div className="overflow-x-auto">
            <Table
              className="w-full min-w-[1150px] table-fixed [&_td]:py-2 [&_th]:py-2"
              aria-busy={isLoading}
            >
              <TableHeader className="sticky top-0 bg-white z-10">
                <TableRow>
                  <TableHead className="w-[50px] bg-slate-50/40 text-center whitespace-nowrap">
                    No
                  </TableHead>
                  <TableHead
                    className="w-[190px] bg-slate-50/40 whitespace-nowrap"
                    aria-sort={sortState("nama")}
                  >
                    <button
                      type="button"
                      className="inline-flex w-full items-center rounded-sm text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      onClick={() => handleSort("nama")}
                    >
                      Nama Pimpinan
                      <SortIcon col="nama" />
                    </button>
                  </TableHead>
                  <TableHead
                    className="w-[110px] bg-slate-50/40 whitespace-nowrap"
                    aria-sort={sortState("organisasi")}
                  >
                    <button
                      type="button"
                      className="inline-flex w-full items-center rounded-sm text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      onClick={() => handleSort("organisasi")}
                    >
                      Organisasi
                      <SortIcon col="organisasi" />
                    </button>
                  </TableHead>
                  <TableHead
                    className="w-[135px] bg-slate-50/40 whitespace-nowrap"
                    aria-sort={sortState("tanggalMulai")}
                  >
                    <button
                      type="button"
                      className="inline-flex w-full items-center rounded-sm text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      onClick={() => handleSort("tanggalMulai")}
                    >
                      Tanggal Mulai
                      <SortIcon col="tanggalMulai" />
                    </button>
                  </TableHead>
                  <TableHead
                    className="w-[135px] bg-slate-50/40 whitespace-nowrap"
                    aria-sort={sortState("tanggalBerakhir")}
                  >
                    <button
                      type="button"
                      className="inline-flex w-full items-center rounded-sm text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      onClick={() => handleSort("tanggalBerakhir")}
                    >
                      Tanggal Berakhir
                      <SortIcon col="tanggalBerakhir" />
                    </button>
                  </TableHead>
                  <TableHead
                    className="w-[190px] bg-slate-50/40 whitespace-nowrap"
                    aria-sort={sortState("status")}
                  >
                    <button
                      type="button"
                      className="inline-flex w-full items-center rounded-sm text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      onClick={() => handleSort("status")}
                    >
                      Status
                      <SortIcon col="status" />
                    </button>
                  </TableHead>
                  <TableHead
                    className="w-[210px] bg-slate-50/40 whitespace-nowrap"
                    aria-sort={sortState("catatan")}
                  >
                    <button
                      type="button"
                      className="inline-flex w-full items-center rounded-sm text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
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
                {sortedData.filter(
                  (item) => !optimisticHiddenIds.includes(item.id),
                ).length === 0 ? (
                  <TableRow>
                    <TableCell
                      colSpan={8}
                      className="h-32 text-center text-muted-foreground"
                    >
                      {searchTerm || orgFilter !== "ALL"
                        ? "Tidak ada data berkas SP yang cocok dengan filter."
                        : "Belum ada data berkas SP."}
                    </TableCell>
                  </TableRow>
                ) : (
                  sortedData
                    .filter((item) => !optimisticHiddenIds.includes(item.id))
                    .map((berkas, index) => (
                      <TableRow key={berkas.id}>
                        <TableCell className="text-center text-muted-foreground font-medium whitespace-nowrap">
                          {(currentPage - 1) * 10 + index + 1}
                        </TableCell>
                        <TableCell
                          className="font-medium whitespace-nowrap"
                          title={berkas.nama}
                        >
                          <div className="truncate">
                            {capitalizeName(berkas.nama)}
                          </div>
                        </TableCell>
                        <TableCell className="whitespace-nowrap">
                          {berkas.organisasi && (
                            <Badge
                              variant="outline"
                              className={cn(
                                "font-medium whitespace-nowrap",
                                organisasiConfig[berkas.organisasi]
                                  ?.className ||
                                  "bg-slate-50 text-slate-700 border-slate-200",
                              )}
                            >
                              {organisasiConfig[berkas.organisasi]?.label ||
                                berkas.organisasi}
                            </Badge>
                          )}
                        </TableCell>
                        <TableCell className="whitespace-nowrap pr-6">
                          {new Date(berkas.tanggalMulai).toLocaleDateString(
                            "id-ID",
                            {
                              day: "2-digit",
                              month: "short",
                              year: "numeric",
                            },
                          )}
                        </TableCell>
                        <TableCell className="whitespace-nowrap pr-6">
                          {new Date(berkas.tanggalBerakhir).toLocaleDateString(
                            "id-ID",
                            {
                              day: "2-digit",
                              month: "short",
                              year: "numeric",
                            },
                          )}
                        </TableCell>
                        <TableCell className="whitespace-nowrap">
                          {getStatusBadge(berkas.tanggalBerakhir)}
                        </TableCell>
                        <TableCell className="whitespace-nowrap">
                          <div
                            className="truncate"
                            title={berkas.catatan || ""}
                          >
                            {berkas.catatan
                              ? capitalizeName(berkas.catatan)
                              : "-"}
                          </div>
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex items-center justify-end gap-2">
                            <Button
                              size="sm"
                              variant="outline"
                              className="h-11 w-11 p-0 sm:h-8 sm:w-8"
                              asChild
                              title={`Lihat detail ${berkas.nama}`}
                            >
                              <Link
                                href={`/dashboard/berkas-sp/${berkas.id}`}
                                aria-label={`Lihat detail Berkas SP ${berkas.nama}`}
                              >
                                <Eye className="w-4 h-4" />
                              </Link>
                            </Button>

                            <Button
                              size="sm"
                              variant="outline"
                              className="h-11 w-11 p-0 sm:h-8 sm:w-8"
                              asChild
                              title={`Edit ${berkas.nama}`}
                            >
                              <Link
                                href={`/dashboard/berkas-sp/${berkas.id}/edit`}
                                aria-label={`Edit Berkas SP ${berkas.nama}`}
                              >
                                <Pencil className="w-4 h-4" />
                              </Link>
                            </Button>
                            <Button
                              size="sm"
                              variant="outline"
                              className="h-11 w-11 p-0 text-red-500 hover:text-red-600 sm:h-8 sm:w-8"
                              onClick={() => setConfirmDeleteId(berkas.id)}
                              title={`Hapus ${berkas.nama}`}
                              aria-label={`Hapus Berkas SP ${berkas.nama}`}
                            >
                              <Trash2 className="w-4 h-4" />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))
                )}

                {/* Pagination Row - integrated into table body */}
                {totalPages >= 1 && (
                  <TableRow className="hover:bg-transparent border-t bg-white">
                    <TableCell colSpan={8} className="p-0">
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
        onClose={() => setConfirmDeleteId(null)}
        onConfirm={handleDelete}
        title="Hapus Berkas SP?"
        description={`Apakah Anda yakin ingin menghapus berkas SP milik "${capitalizeName(data.find((b) => b.id === confirmDeleteId)?.nama || "")}"? Tindakan ini tidak dapat dibatalkan dan file terkait akan dihapus secara permanen.`}
        variant="destructive"
        loading={false}
      />
    </div>
  );
}
