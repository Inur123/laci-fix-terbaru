"use client";

import React, { useState, useEffect, useRef } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
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
  Eye,
  Pencil,
  Trash2,
  Search,
  FileSpreadsheet,
  ArrowUpDown,
  ArrowUp,
  ArrowDown,
} from "lucide-react";
import { deleteArsipSurat, getArsipSurats } from "@/app/actions/arsip-actions";
import { logExport } from "@/app/actions/log-activity-actions";
import { ConfirmModal } from "@/components/shared/confirm-modal";
import { toast } from "sonner";
import { Label } from "@/components/ui/label";
import XLSX from "@/lib/excel";
import { fetchAllPages } from "@/lib/fetch-all-pages";
import { cn } from "@/lib/utils";

import { DecryptedArsipSurat } from "@/types";

export function ArsipSuratList({
  arsipSurats: initialArsipSurats,
  userRole,
  totalPages: initialTotalPages,
  currentPage: initialCurrentPage,
  totalItems: initialTotalItems,
}: {
  arsipSurats: DecryptedArsipSurat[];
  userRole: string;
  totalPages: number;
  currentPage: number;
  totalItems: number;
}) {
  // Local data state
  const [data, setData] = useState<DecryptedArsipSurat[]>(initialArsipSurats);
  const [totalPages, setTotalPages] = useState(initialTotalPages);
  const [currentPage, setCurrentPage] = useState(initialCurrentPage);
  const [totalItems, setTotalItems] = useState(initialTotalItems);
  const [isLoading, setIsLoading] = useState(false);

  // Filter state
  const [searchTerm, setSearchTerm] = useState("");
  const [orgFilter, setOrgFilter] = useState("ALL");
  const [jenisFilter, setJenisFilter] = useState("ALL");

  // Sort state
  type SortKey =
    | "tanggal"
    | "noSurat"
    | "pengirimPenerima"
    | "perihal"
    | "organisasi"
    | "jenisSurat";
  type SortDir = "asc" | "desc";
  const [sortKey, setSortKey] = useState<SortKey | null>(null);
  const [sortDir, setSortDir] = useState<SortDir>("desc");

  // Function to fetch data with sorting parameters
  const isFirstRender = useRef(true);

  const fetchData = async (
    query: string,
    org: string,
    jenis: string,
    page: number,
    sKey: SortKey | null = sortKey,
    sDir: SortDir = sortDir,
  ) => {
    setIsLoading(true);
    try {
      const result = await getArsipSurats(
        query,
        org,
        jenis,
        page,
        10,
        sKey,
        sDir,
      );
      setData(result.data as DecryptedArsipSurat[]);
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
    fetchData(searchTerm, orgFilter, jenisFilter, 1, key, newDir);
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

  // Debounced Search Update
  useEffect(() => {
    if (isFirstRender.current) {
      isFirstRender.current = false;
      return;
    }
    const timer = setTimeout(() => {
      setCurrentPage(1);
      fetchData(searchTerm, orgFilter, jenisFilter, 1);
    }, 500);

    return () => clearTimeout(timer);
  }, [searchTerm, orgFilter, jenisFilter]);

  useEffect(() => {
    const handler = (event: Event) => {
      const detail = (event as CustomEvent).detail as {
        type?: string;
        model?: string;
      };
      if (!detail || detail.type !== "mutation") return;
      if (detail.model !== "ArsipSurat") return;
      if (realtimeTimerRef.current) return;
      realtimeTimerRef.current = setTimeout(() => {
        realtimeTimerRef.current = null;
        fetchData(searchTerm, orgFilter, jenisFilter, currentPage);
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
  }, [searchTerm, orgFilter, jenisFilter, currentPage]);

  const handleFilterChange = (key: string, value: string) => {
    if (key === "org") {
      setOrgFilter(value);
    } else if (key === "jenis") {
      setJenisFilter(value);
    }

    setCurrentPage(1);
  };

  const handlePageChange = (page: number) => {
    setCurrentPage(page);
    fetchData(searchTerm, orgFilter, jenisFilter, page);
  };

  const handleDelete = async () => {
    if (!confirmDeleteId) return;
    const id = confirmDeleteId;
    setConfirmDeleteId(null);
    setOptimisticHiddenIds((prev) => [...prev, id]);

    const result = await deleteArsipSurat(id);

    if (result.error) {
      setOptimisticHiddenIds((prev) => prev.filter((pid) => pid !== id));
      toast.error(result.error);
    } else {
      toast.success("Arsip surat berhasil dihapus");
      window.dispatchEvent(
        new CustomEvent("laci-realtime", {
          detail: { type: "mutation", model: "ArsipSurat" },
        }),
      );
    }
  };

  const handleExportExcel = async () => {
    if (totalItems === 0) {
      toast.error("Tidak ada data untuk diexport");
      return;
    }

    toast.info("Menyiapkan data export...");

    const suffix = userRole === "SEKRETARIS_CABANG" ? "Cabang" : "PAC";
    const dateStr = new Date().toLocaleDateString("id-ID").replace(/\//g, "-");
    const filename = `Arsip_Surat_${suffix}_${dateStr}.xlsx`;

    let allData = data;
    if (totalItems > data.length) {
      try {
        allData = await fetchAllPages<DecryptedArsipSurat>((page, limit) =>
          getArsipSurats(
            searchTerm,
            orgFilter,
            jenisFilter,
            page,
            limit,
            sortKey,
            sortDir,
          ),
        );
      } catch {
        toast.error("Gagal mengambil semua data untuk export");
        return;
      }
    }

    const exportData: Record<string, string | number>[] = allData.map(
      (item, index) => ({
        No: index + 1,
        "No. Surat": item.noSurat,
        "Jenis Surat": item.jenisSurat,
        Organisasi: item.organisasi
          ? organisasiConfig[item.organisasi]?.label || item.organisasi
          : "-",
        Tanggal: new Date(item.tanggal).toLocaleDateString("id-ID", {
          day: "numeric",
          month: "long",
          year: "numeric",
        }),
        "Pengirim/Penerima": item.pengirimPenerima,
        Perihal: item.perihal,
        Deskripsi: item.deskripsi || "-",
      }),
    );

    const worksheet = XLSX.utils.json_to_sheet(exportData);
    const range = XLSX.utils.decode_range(worksheet["!ref"] || "A1:H1");
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
    XLSX.utils.book_append_sheet(workbook, worksheet, "Arsip Surat");

    const wscols = Object.keys(exportData[0] || {}).map((key) => {
      const maxLen = Math.max(
        key.length,
        ...exportData.map((row) => String(row[key] || "").length),
      );
      return { wch: Math.min(maxLen + 2, 50) };
    });
    worksheet["!cols"] = wscols;

    await XLSX.writeFile(workbook, filename);
    logExport("ARSIP_SURAT", filename); // Fire and forget
    toast.success("File excel berhasil didownload!");
  };

  const organisasiConfig: Record<string, { label: string; className: string }> =
    {
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
      BERSAMA: {
        label: "BERSAMA",
        className:
          "bg-indigo-100/80 text-indigo-700 border-indigo-200 hover:bg-indigo-200/80",
      },
      CBP_KPP: {
        label: "CBP_KPP",
        className:
          "bg-orange-100/80 text-orange-700 border-orange-200 hover:bg-orange-200/80",
      },
    };

  const jenisSuratConfig: Record<string, { label: string; className: string }> =
    {
      MASUK: {
        label: "Surat Masuk",
        className:
          "bg-blue-100/80 text-blue-700 border-blue-200 hover:bg-blue-200/80",
      },
      KELUAR: {
        label: "Surat Keluar",
        className:
          "bg-amber-100/80 text-amber-700 border-amber-200 hover:bg-amber-200/80",
      },
    };

  return (
    <div className="flex flex-col">
      {/* Filter Section */}
      <div className="flex flex-col md:flex-row gap-4 mb-4 items-end">
        {/* Search */}
        <div className="flex-1 relative w-full">
          <Label
            htmlFor="arsip-search"
            className="text-xs font-medium mb-1 block"
          >
            Cari Arsip
          </Label>
          <div className="relative">
            <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              id="arsip-search"
              placeholder="Cari nomor surat, perihal, atau pengirim..."
              className="pl-9 w-full bg-white h-9 text-sm"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
            />
          </div>
        </div>

        {/* Organisasi */}
        <div className="w-full md:w-44">
          <Label
            htmlFor="arsip-organisasi-filter"
            className="text-xs font-medium mb-1 block"
          >
            Organisasi
          </Label>
          <Select
            value={orgFilter}
            onValueChange={(val) => handleFilterChange("org", val)}
          >
            <SelectTrigger
              id="arsip-organisasi-filter"
              aria-label="Filter organisasi arsip"
              className="w-full bg-white h-9 text-sm"
            >
              <SelectValue placeholder="Semua" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="ALL">Semua</SelectItem>
              <SelectItem value="IPNU">IPNU</SelectItem>
              <SelectItem value="IPPNU">IPPNU</SelectItem>
              <SelectItem value="BERSAMA">BERSAMA</SelectItem>
              <SelectItem value="CBP_KPP">CBP KPP</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {/* Jenis */}
        <div className="w-full md:w-36">
          <Label
            htmlFor="arsip-jenis-filter"
            className="text-xs font-medium mb-1 block"
          >
            Jenis
          </Label>
          <Select
            value={jenisFilter}
            onValueChange={(val) => handleFilterChange("jenis", val)}
          >
            <SelectTrigger
              id="arsip-jenis-filter"
              aria-label="Filter jenis arsip"
              className="w-full bg-white h-9 text-sm"
            >
              <SelectValue placeholder="Semua" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="ALL">Semua</SelectItem>
              <SelectItem value="MASUK">Masuk</SelectItem>
              <SelectItem value="KELUAR">Keluar</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="w-full md:w-auto">
          <Button
            variant="outline"
            className="h-9 w-full md:w-auto px-4 text-sm bg-white border-slate-200 shadow-sm"
            onClick={handleExportExcel}
            disabled={isLoading}
            aria-label="Export arsip surat ke Excel"
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
              className="w-full min-w-[1120px] table-fixed"
              aria-busy={isLoading}
            >
              <TableHeader className="sticky top-0 bg-white z-10">
                <TableRow>
                  <TableHead className="w-[50px] bg-slate-50/40 text-center whitespace-nowrap">
                    No
                  </TableHead>
                  <TableHead
                    className="w-[120px] bg-slate-50/40 whitespace-nowrap"
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
                    className="w-[160px] bg-slate-50/40 whitespace-nowrap"
                    aria-sort={sortState("noSurat")}
                  >
                    <button
                      type="button"
                      className="inline-flex w-full items-center rounded-sm text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      onClick={() => handleSort("noSurat")}
                    >
                      No. Surat
                      <SortIcon col="noSurat" />
                    </button>
                  </TableHead>
                  <TableHead
                    className="w-[100px] bg-slate-50/40 whitespace-nowrap"
                    aria-sort={sortState("jenisSurat")}
                  >
                    <button
                      type="button"
                      className="inline-flex w-full items-center rounded-sm text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      onClick={() => handleSort("jenisSurat")}
                    >
                      Jenis
                      <SortIcon col="jenisSurat" />
                    </button>
                  </TableHead>
                  <TableHead
                    className="w-[150px] bg-slate-50/40 whitespace-nowrap"
                    aria-sort={sortState("tanggal")}
                  >
                    <button
                      type="button"
                      className="inline-flex w-full items-center rounded-sm text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      onClick={() => handleSort("tanggal")}
                    >
                      Tanggal
                      <SortIcon col="tanggal" />
                    </button>
                  </TableHead>
                  <TableHead
                    className="w-[180px] bg-slate-50/40 whitespace-nowrap"
                    aria-sort={sortState("pengirimPenerima")}
                  >
                    <button
                      type="button"
                      className="inline-flex w-full items-center rounded-sm text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      onClick={() => handleSort("pengirimPenerima")}
                    >
                      Pengirim/Penerima
                      <SortIcon col="pengirimPenerima" />
                    </button>
                  </TableHead>
                  <TableHead
                    className="bg-slate-50/40 whitespace-nowrap"
                    aria-sort={sortState("perihal")}
                  >
                    <button
                      type="button"
                      className="inline-flex w-full items-center rounded-sm text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      onClick={() => handleSort("perihal")}
                    >
                      Perihal
                      <SortIcon col="perihal" />
                    </button>
                  </TableHead>
                  <TableHead className="w-[130px] text-right bg-slate-50/40 whitespace-nowrap">
                    Aksi
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {sortedData.filter(
                  (item: DecryptedArsipSurat) =>
                    !optimisticHiddenIds.includes(item.id),
                ).length === 0 ? (
                  <TableRow>
                    <TableCell
                      colSpan={8}
                      className="h-32 text-center text-muted-foreground"
                    >
                      {searchTerm ||
                      orgFilter !== "ALL" ||
                      jenisFilter !== "ALL"
                        ? "Tidak ada data arsip surat yang cocok dengan filter."
                        : "Belum ada data arsip surat."}
                    </TableCell>
                  </TableRow>
                ) : (
                  sortedData
                    .filter(
                      (item: DecryptedArsipSurat) =>
                        !optimisticHiddenIds.includes(item.id),
                    )
                    .map((arsip: DecryptedArsipSurat, index: number) => (
                      <TableRow key={arsip.id}>
                        <TableCell className="text-center text-muted-foreground font-medium whitespace-nowrap">
                          {(currentPage - 1) * 10 + index + 1}
                        </TableCell>
                        <TableCell className="whitespace-nowrap">
                          {arsip.organisasi ? (
                            <Badge
                              variant="outline"
                              className={cn(
                                "whitespace-nowrap transition-colors",
                                organisasiConfig[arsip.organisasi]?.className ||
                                  "",
                              )}
                            >
                              {arsip.organisasi}
                            </Badge>
                          ) : (
                            "-"
                          )}
                        </TableCell>
                        <TableCell
                          className="max-w-[160px] truncate font-medium"
                          title={arsip.noSurat}
                        >
                          {arsip.noSurat}
                        </TableCell>
                        <TableCell className="whitespace-nowrap">
                          <Badge
                            variant="outline"
                            className={cn(
                              "whitespace-nowrap transition-colors",
                              jenisSuratConfig[arsip.jenisSurat]?.className ||
                                "",
                            )}
                          >
                            {arsip.jenisSurat}
                          </Badge>
                        </TableCell>
                        <TableCell className="whitespace-nowrap pr-6">
                          {new Date(arsip.tanggal).toLocaleDateString("id-ID", {
                            day: "numeric",
                            month: "long",
                            year: "numeric",
                          })}
                        </TableCell>
                        <TableCell
                          className="max-w-[180px] truncate"
                          title={arsip.pengirimPenerima}
                        >
                          {arsip.pengirimPenerima}
                        </TableCell>
                        <TableCell
                          className="max-w-[300px] truncate whitespace-nowrap"
                          title={arsip.perihal}
                        >
                          {arsip.perihal}
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex items-center justify-end gap-2">
                            <Button
                              size="sm"
                              variant="outline"
                              className="h-11 w-11 p-0 sm:h-8 sm:w-8"
                              asChild
                              title={`Lihat detail ${arsip.noSurat}`}
                            >
                              <Link
                                href={`/dashboard/arsip/surat/${arsip.id}`}
                                aria-label={`Lihat detail arsip ${arsip.noSurat}`}
                              >
                                <Eye className="w-4 h-4" />
                              </Link>
                            </Button>
                            <Button
                              size="sm"
                              variant="outline"
                              className="h-11 w-11 p-0 sm:h-8 sm:w-8"
                              asChild
                              title={`Edit ${arsip.noSurat}`}
                            >
                              <Link
                                href={`/dashboard/arsip/surat/${arsip.id}/edit`}
                                aria-label={`Edit arsip ${arsip.noSurat}`}
                              >
                                <Pencil className="w-4 h-4" />
                              </Link>
                            </Button>
                            <Button
                              size="sm"
                              variant="outline"
                              className="h-11 w-11 p-0 text-red-500 hover:text-red-600 sm:h-8 sm:w-8"
                              onClick={() => setConfirmDeleteId(arsip.id)}
                              title={`Hapus ${arsip.noSurat}`}
                              aria-label={`Hapus arsip ${arsip.noSurat}`}
                            >
                              <Trash2 className="w-4 h-4" />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))
                )}

                {/* Pagination Row – integrated into table body */}
                {totalPages >= 1 && (
                  <TableRow className="hover:bg-transparent border-t bg-slate-50/30">
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
                          arsip
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
        title="Hapus Arsip Surat?"
        description="Apakah Anda yakin ingin menghapus arsip surat ini? Tindakan ini tidak dapat dibatalkan dan file terkait akan dihapus secara permanen."
        variant="destructive"
        loading={false}
      />
    </div>
  );
}
