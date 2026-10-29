"use client";

import { useState, useEffect, useRef, useCallback } from "react";
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
import { useRouter, useSearchParams } from "next/navigation";
import {
  Eye,
  Pencil,
  Trash2,
  Search,
  User as UserIcon,
  RefreshCcw,
  ArrowUpDown,
  ArrowUp,
  ArrowDown,
} from "lucide-react";
import { deleteAnggota, getAnggotaList } from "@/app/actions/anggota-actions";
import { logExport } from "@/app/actions/log-activity-actions";
import { toast } from "sonner";
import { Label } from "@/components/ui/label";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { cn, capitalizeName } from "@/lib/utils";
import { FileSpreadsheet } from "lucide-react";
import XLSX from "@/lib/excel";
import { fetchAllPages } from "@/lib/fetch-all-pages";
import { Spinner } from "@/components/ui/spinner";

type AnggotaItem = {
  id: string;
  namaLengkap: string;
  nik?: string | null;
  nia?: string | null;
  jenisKelamin?: string | null;
  tempatLahir?: string | null;
  tanggalLahir?: Date | string | null;
  alamatLengkap?: string | null;
  noHp?: string | null;
  email?: string | null;
  jabatan?: string | null;
  noRfid?: string | null;
  pekerjaan?: string | null;
  foto?: string | null;
  updatedAt?: Date | string | null;
  pendidikans?: Array<{
    jenjang: string;
    namaSekolah: string;
  }>;
  user?: { name?: string | null } | null;
  periode?: { nama?: string | null } | null;
  perkaderans?: Array<{
    namaPerkaderan: string;
    tanggal: Date | string;
    tempat: string;
  }>;
};

type SortKey =
  | "createdAt"
  | "namaLengkap"
  | "jabatan"
  | "jenisKelamin"
  | "noHp"
  | "periode"
  | "dibuatOleh";
type SortDir = "asc" | "desc";

import { UserFilterSelect } from "@/components/shared/user-filter-select";
import { ConfirmModal } from "@/components/shared/confirm-modal";

export function AnggotaList({
  anggotaList: initialAnggotaList,
  userRole,
  totalPages: initialTotalPages,
  currentPage: initialCurrentPage,
  totalItems: initialTotalItems,
  activeUsers,
  initialSearchTerm = "",
  initialSelectedUser = "ALL",
  initialSortKey = "createdAt",
  initialSortDir = "desc",
}: {
  anggotaList: AnggotaItem[];
  userRole: string;
  totalPages: number;
  currentPage: number;
  totalItems: number;
  activeUsers?: { id: string; name: string }[];
  initialSearchTerm?: string;
  initialSelectedUser?: string;
  initialSortKey?: string;
  initialSortDir?: string;
}) {
  const router = useRouter();
  const searchParams = useSearchParams();

  const [data, setData] = useState<AnggotaItem[]>(initialAnggotaList);
  const [totalPages, setTotalPages] = useState(initialTotalPages);
  const [currentPage, setCurrentPage] = useState(initialCurrentPage);
  const [totalItems, setTotalItems] = useState(initialTotalItems);
  const [isLoading, setIsLoading] = useState(false);
  const [loadError, setLoadError] = useState("");

  useEffect(() => {
    setData(initialAnggotaList);
    setTotalPages(initialTotalPages);
    setCurrentPage(initialCurrentPage);
    setTotalItems(initialTotalItems);
  }, [
    initialAnggotaList,
    initialTotalPages,
    initialCurrentPage,
    initialTotalItems,
  ]);

  const [searchTerm, setSearchTerm] = useState(initialSearchTerm);
  const [selectedUser, setSelectedUser] = useState(initialSelectedUser);

  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const realtimeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const [sortKey, setSortKey] = useState<SortKey | null>(
    (initialSortKey as SortKey) || "createdAt",
  );
  const [sortDir, setSortDir] = useState<SortDir>(
    initialSortDir === "asc" ? "asc" : "desc",
  );

  useEffect(() => {
    setSearchTerm(initialSearchTerm);
    setSelectedUser(initialSelectedUser);
    setSortKey((initialSortKey as SortKey) || "createdAt");
    setSortDir(initialSortDir === "asc" ? "asc" : "desc");
  }, [initialSearchTerm, initialSelectedUser, initialSortKey, initialSortDir]);

  const handleSort = (key: SortKey) => {
    let nextDir: SortDir = "asc";
    if (sortKey === key) {
      nextDir = sortDir === "asc" ? "desc" : "asc";
      setSortDir(nextDir);
    } else {
      setSortKey(key);
      setSortDir("asc");
    }
    setCurrentPage(1);
    fetchData(searchTerm, 1, selectedUser, key, nextDir);
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

  const sortedData = data;

  const isFirstRender = useRef(true);

  const fetchData = useCallback(
    async (
      query: string,
      page: number,
      userId: string,
      sKey: SortKey | null,
      sDir: SortDir,
    ) => {
      setIsLoading(true);
      setLoadError("");
      try {
        const result = await getAnggotaList(
          query,
          page,
          10,
          userId,
          undefined,
          sKey,
          sDir,
        );
        setData(result.data as AnggotaItem[]);
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
          setLoadError("Data anggota gagal dimuat. Silakan coba lagi.");
          toast.error(`Gagal memuat data: ${errMsg}`);
        }
      } finally {
        setIsLoading(false);
      }
    },
    [],
  );

  useEffect(() => {
    if (isFirstRender.current) {
      isFirstRender.current = false;
      return;
    }
    const timer = setTimeout(() => {
      setCurrentPage(1);
      fetchData(searchTerm, 1, selectedUser, sortKey, sortDir);
    }, 500);

    return () => clearTimeout(timer);
  }, [searchTerm, selectedUser, sortKey, sortDir, fetchData]);

  useEffect(() => {
    const handler = (event: Event) => {
      const detail = (event as CustomEvent).detail as {
        type?: string;
        model?: string;
      };
      if (!detail || detail.type !== "mutation") return;
      if (detail.model !== "Anggota") return;
      if (realtimeTimerRef.current) return;
      realtimeTimerRef.current = setTimeout(() => {
        realtimeTimerRef.current = null;
        fetchData(searchTerm, currentPage, selectedUser, sortKey, sortDir);
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
  }, [searchTerm, currentPage, selectedUser, sortKey, sortDir, fetchData]);

  const handleUserFilterChange = (value: string) => {
    setSelectedUser(value);
    setCurrentPage(1);

    const params = new URLSearchParams(searchParams.toString());
    if (value === "ALL") {
      params.delete("userId");
    } else {
      params.set("userId", value);
    }
    params.set("page", "1");
    router.push(`?${params.toString()}`, { scroll: false });
  };

  const handleResetFilters = () => {
    setSearchTerm("");
    setSelectedUser("ALL");
    setCurrentPage(1);

    const params = new URLSearchParams(searchParams.toString());
    params.delete("userId");
    params.delete("q");
    params.set("page", "1");
    router.push(`?${params.toString()}`, { scroll: false });
  };

  const handlePageChange = (page: number) => {
    setCurrentPage(page);
    fetchData(searchTerm, page, selectedUser, sortKey, sortDir);
  };

  const handleDelete = async () => {
    if (!confirmDeleteId || isDeleting) return;
    const id = confirmDeleteId;
    setIsDeleting(true);
    try {
      const result = await deleteAnggota(id);
      if (result.error) {
        toast.error(result.error);
        return;
      }

      setConfirmDeleteId(null);
      toast.success("Anggota berhasil dihapus");
      await fetchData(searchTerm, currentPage, selectedUser, sortKey, sortDir);
    } finally {
      setIsDeleting(false);
    }
  };

  const getInitials = (name: string) => {
    return name
      .split(" ")
      .map((n) => n[0])
      .join("")
      .toUpperCase()
      .slice(0, 2);
  };

  const handleExportExcel = async () => {
    if (totalItems === 0) {
      toast.error("Tidak ada data untuk diexport");
      return;
    }

    toast.info("Menyiapkan data export...");

    const dateStr = new Date().toLocaleDateString("id-ID").replace(/\//g, "-");

    let userName = "All";
    if (selectedUser !== "ALL" && activeUsers) {
      const user = activeUsers.find((u) => u.id === selectedUser);
      if (user) userName = user.name;
    }
    const safeUserName = userName.replace(/[^a-zA-Z0-9]/g, "_");
    const filename = `Data_Anggota_${safeUserName}_${dateStr}.xlsx`;

    let allData: AnggotaItem[];
    try {
      allData = await fetchAllPages<AnggotaItem>((page, limit) =>
        getAnggotaList(
          searchTerm,
          page,
          limit,
          selectedUser,
          undefined,
          sortKey,
          sortDir,
          true,
        ),
      );
    } catch {
      toast.error("Gagal mengambil semua data untuk export");
      return;
    }

    const exportData: Record<string, string | number>[] = allData.map(
      (item, index) => ({
        No: index + 1,
        Nama: item.namaLengkap,
        NIK: item.nik || "-",
        NIA: item.nia || "-",
        "Jenis Kelamin":
          item.jenisKelamin === "LAKI_LAKI" ? "Laki-laki" : "Perempuan",
        "Tempat Lahir": item.tempatLahir || "-",
        "Tanggal Lahir": item.tanggalLahir
          ? new Date(item.tanggalLahir).toLocaleDateString("id-ID", {
              day: "numeric",
              month: "long",
              year: "numeric",
            })
          : "-",
        Alamat: item.alamatLengkap || "-",
        "No HP": item.noHp || "-",
        Email: item.email || "-",
        Jabatan: item.jabatan || "-",
        Pekerjaan: item.pekerjaan || "-",
        SD:
          item.pendidikans?.find((p) => p.jenjang === "SD")?.namaSekolah || "-",
        MI:
          item.pendidikans?.find((p) => p.jenjang === "MI")?.namaSekolah || "-",
        SMP:
          item.pendidikans?.find((p) => p.jenjang === "SMP")?.namaSekolah ||
          "-",
        MTs:
          item.pendidikans?.find((p) => p.jenjang === "MTs")?.namaSekolah ||
          "-",
        SMA:
          item.pendidikans?.find((p) => p.jenjang === "SMA")?.namaSekolah ||
          "-",
        SMK:
          item.pendidikans?.find((p) => p.jenjang === "SMK")?.namaSekolah ||
          "-",
        MAN:
          item.pendidikans?.find((p) => p.jenjang === "MAN")?.namaSekolah ||
          "-",
        KULIAH:
          item.pendidikans?.find((p) => p.jenjang === "KULIAH")?.namaSekolah ||
          "-",
        Makesta:
          item.perkaderans
            ?.filter((p) => p.namaPerkaderan.toUpperCase() === "MAKESTA")
            .map(
              (p) =>
                `${new Date(p.tanggal).toLocaleDateString("id-ID", {
                  day: "numeric",
                  month: "long",
                  year: "numeric",
                })} (${p.tempat})`,
            )
            .join(", ") || "-",
        Lakmud:
          item.perkaderans
            ?.filter((p) => p.namaPerkaderan.toUpperCase() === "LAKMUD")
            .map(
              (p) =>
                `${new Date(p.tanggal).toLocaleDateString("id-ID", {
                  day: "numeric",
                  month: "long",
                  year: "numeric",
                })} (${p.tempat})`,
            )
            .join(", ") || "-",
        Lakut:
          item.perkaderans
            ?.filter((p) => p.namaPerkaderan.toUpperCase() === "LAKUT")
            .map(
              (p) =>
                `${new Date(p.tanggal).toLocaleDateString("id-ID", {
                  day: "numeric",
                  month: "long",
                  year: "numeric",
                })} (${p.tempat})`,
            )
            .join(", ") || "-",
        Diklatama:
          item.perkaderans
            ?.filter((p) => p.namaPerkaderan.toUpperCase() === "DIKLATAMA")
            .map(
              (p) =>
                `${new Date(p.tanggal).toLocaleDateString("id-ID", {
                  day: "numeric",
                  month: "long",
                  year: "numeric",
                })} (${p.tempat})`,
            )
            .join(", ") || "-",
        Diklatmad:
          item.perkaderans
            ?.filter((p) => p.namaPerkaderan.toUpperCase() === "DIKLATMAD")
            .map(
              (p) =>
                `${new Date(p.tanggal).toLocaleDateString("id-ID", {
                  day: "numeric",
                  month: "long",
                  year: "numeric",
                })} (${p.tempat})`,
            )
            .join(", ") || "-",
        Latin:
          item.perkaderans
            ?.filter((p) => p.namaPerkaderan.toUpperCase() === "LATIN")
            .map(
              (p) =>
                `${new Date(p.tanggal).toLocaleDateString("id-ID", {
                  day: "numeric",
                  month: "long",
                  year: "numeric",
                })} (${p.tempat})`,
            )
            .join(", ") || "-",
        Latpel:
          item.perkaderans
            ?.filter((p) => p.namaPerkaderan.toUpperCase() === "LATPEL")
            .map(
              (p) =>
                `${new Date(p.tanggal).toLocaleDateString("id-ID", {
                  day: "numeric",
                  month: "long",
                  year: "numeric",
                })} (${p.tempat})`,
            )
            .join(", ") || "-",
        "No RFID": item.noRfid || "-",
        "Dibuat Oleh": item.user?.name || "-",
        Periode: item.periode?.nama || "-",
      }),
    );

    const worksheet = XLSX.utils.json_to_sheet(exportData);
    const range = XLSX.utils.decode_range(worksheet["!ref"] || "A1:N1");
    const headerColor = "3b82f6";

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
    XLSX.utils.book_append_sheet(workbook, worksheet, "Data Anggota");
    await XLSX.writeFile(workbook, filename);
    void logExport("ANGGOTA", filename);
    toast.success("File excel berhasil didownload!");
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col md:flex-row gap-4 mb-4 items-end">
        <div className="flex-1 w-full relative">
          <Label
            htmlFor="anggota-search"
            className="text-xs font-medium mb-1 block"
          >
            Cari Anggota
          </Label>
          <div className="relative">
            <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              id="anggota-search"
              placeholder="Cari nama, jabatan, NIK, atau NIA..."
              className="min-h-11 pl-9 w-full bg-white text-sm border-slate-200 shadow-sm"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
            />
          </div>
        </div>

        {activeUsers && activeUsers.length > 0 && (
          <div className="w-full md:w-64">
            <Label className="text-xs font-medium mb-1 block">
              Filter User
            </Label>
            <UserFilterSelect
              users={activeUsers}
              selectedUserId={selectedUser}
              onSelectUser={handleUserFilterChange}
              ariaLabel="Filter pembuat data anggota"
              className="min-h-11"
            />
          </div>
        )}

        <div className="grid grid-cols-2 gap-2 w-full md:flex md:w-auto md:items-center md:justify-end md:gap-3 lg:justify-start">
          <Button
            variant="outline"
            aria-label={`Export ${totalItems} data anggota ke Excel`}
            className="min-h-11 w-full md:w-auto px-4 text-sm bg-white border-slate-200 shadow-sm whitespace-nowrap text-slate-600 hover:text-slate-900"
            onClick={handleExportExcel}
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

          <Button
            variant="outline"
            className={cn(
              "min-h-11 w-full md:w-auto px-4 text-sm bg-white border-slate-200 shadow-sm whitespace-nowrap transition-all duration-200",
              searchTerm !== "" || selectedUser !== "ALL"
                ? "text-slate-900 border-slate-300 opacity-100"
                : "text-slate-400 border-slate-200 opacity-50 cursor-not-allowed",
            )}
            onClick={handleResetFilters}
            disabled={searchTerm === "" && selectedUser === "ALL"}
          >
            <RefreshCcw className="mr-2 h-3.5 w-3.5" />
            Reset
          </Button>
        </div>
      </div>

      <div className="relative" aria-busy={isLoading}>
        {isLoading && (
          <div className="absolute inset-0 z-20 flex items-center justify-center rounded-md bg-white/70">
            <Spinner className="h-6 w-6" />
            <span className="sr-only">Memuat data anggota</span>
          </div>
        )}
        <div className="rounded-md border">
          <div className="relative max-h-[600px] overflow-auto">
            <Table className="min-w-[1190px] table-fixed">
              <TableHeader className="sticky top-0 bg-white z-10 border-b">
                <TableRow>
                  <TableHead className="w-[50px] bg-slate-50/40 text-center whitespace-nowrap text-slate-500 font-semibold h-11">
                    No
                  </TableHead>
                  <TableHead
                    className="w-[280px] bg-slate-50/40 whitespace-nowrap text-slate-500 font-semibold h-11 p-0"
                    aria-sort={
                      sortKey === "namaLengkap"
                        ? sortDir === "asc"
                          ? "ascending"
                          : "descending"
                        : "none"
                    }
                  >
                    <button
                      type="button"
                      aria-label="Urutkan berdasarkan nama anggota"
                      className="inline-flex h-11 w-full items-center px-4 text-left hover:bg-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
                      onClick={() => handleSort("namaLengkap")}
                    >
                      Nama
                      <SortIcon col="namaLengkap" />
                    </button>
                  </TableHead>
                  <TableHead
                    className="w-[190px] bg-slate-50/40 whitespace-nowrap text-slate-500 font-semibold h-11 p-0"
                    aria-sort={
                      sortKey === "periode"
                        ? sortDir === "asc"
                          ? "ascending"
                          : "descending"
                        : "none"
                    }
                  >
                    <button
                      type="button"
                      aria-label="Urutkan berdasarkan periode anggota"
                      className="inline-flex h-11 w-full items-center px-4 text-left hover:bg-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
                      onClick={() => handleSort("periode")}
                    >
                      Periode
                      <SortIcon col="periode" />
                    </button>
                  </TableHead>
                  <TableHead
                    className="w-[150px] bg-slate-50/40 whitespace-nowrap text-slate-500 font-semibold h-11 p-0"
                    aria-sort={
                      sortKey === "jenisKelamin"
                        ? sortDir === "asc"
                          ? "ascending"
                          : "descending"
                        : "none"
                    }
                  >
                    <button
                      type="button"
                      aria-label="Urutkan berdasarkan jenis kelamin"
                      className="inline-flex h-11 w-full items-center px-4 text-left hover:bg-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
                      onClick={() => handleSort("jenisKelamin")}
                    >
                      Jenis Kelamin
                      <SortIcon col="jenisKelamin" />
                    </button>
                  </TableHead>
                  <TableHead
                    className="w-[180px] bg-slate-50/40 whitespace-nowrap text-slate-500 font-semibold h-11 p-0"
                    aria-sort={
                      sortKey === "noHp"
                        ? sortDir === "asc"
                          ? "ascending"
                          : "descending"
                        : "none"
                    }
                  >
                    <button
                      type="button"
                      aria-label="Urutkan berdasarkan nomor HP"
                      className="inline-flex h-11 w-full items-center px-4 text-left hover:bg-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
                      onClick={() => handleSort("noHp")}
                    >
                      No. HP
                      <SortIcon col="noHp" />
                    </button>
                  </TableHead>
                  <TableHead
                    className="w-[160px] bg-slate-50/40 whitespace-nowrap text-slate-500 font-semibold h-11 p-0"
                    aria-sort={
                      sortKey === "dibuatOleh"
                        ? sortDir === "asc"
                          ? "ascending"
                          : "descending"
                        : "none"
                    }
                  >
                    <button
                      type="button"
                      aria-label="Urutkan berdasarkan pembuat data"
                      className="inline-flex h-11 w-full items-center px-4 text-left hover:bg-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
                      onClick={() => handleSort("dibuatOleh")}
                    >
                      Dibuat Oleh
                      <SortIcon col="dibuatOleh" />
                    </button>
                  </TableHead>
                  <TableHead className="w-[140px] bg-slate-50/40 text-right whitespace-nowrap text-slate-500 font-semibold h-11">
                    Aksi
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loadError ? (
                  <TableRow>
                    <TableCell
                      colSpan={7}
                      className="h-32 text-center text-red-600"
                      role="alert"
                    >
                      {loadError}
                    </TableCell>
                  </TableRow>
                ) : sortedData.length === 0 ? (
                  <TableRow>
                    <TableCell
                      colSpan={7}
                      className="h-32 text-center text-muted-foreground"
                    >
                      {searchTerm
                        ? "Tidak ada data anggota yang cocok dengan filter."
                        : "Belum ada data anggota."}
                    </TableCell>
                  </TableRow>
                ) : (
                  sortedData.map((item, index) => (
                    <TableRow
                      key={item.id}
                      className="hover:bg-slate-50/50 transition-colors"
                    >
                      <TableCell className="text-center text-slate-500 font-medium whitespace-nowrap">
                        {(currentPage - 1) * 10 + index + 1}
                      </TableCell>
                      <TableCell className="max-w-[280px] overflow-hidden whitespace-nowrap">
                        <div className="flex min-w-0 items-center gap-3">
                          <Avatar className="h-9 w-9 shrink-0 border border-slate-100 shadow-sm">
                            <AvatarImage
                              src={
                                item.foto
                                  ? `/api/anggota/${item.id}/image?v=${item.updatedAt}`
                                  : ""
                              }
                              className="object-cover"
                            />
                            <AvatarFallback className="bg-primary/5 text-primary text-[10px] font-bold">
                              {getInitials(item.namaLengkap)}
                            </AvatarFallback>
                          </Avatar>
                          <div className="min-w-0 flex-1">
                            <span
                              className="block truncate text-sm font-semibold text-slate-900"
                              title={capitalizeName(item.namaLengkap)}
                            >
                              {capitalizeName(item.namaLengkap)}
                            </span>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell className="overflow-hidden whitespace-nowrap">
                        <Badge
                          variant="secondary"
                          className="block w-full max-w-full truncate align-middle font-medium bg-slate-100 text-slate-600 hover:bg-slate-200 border-slate-200"
                          title={item.periode?.nama || "-"}
                        >
                          {item.periode?.nama || "-"}
                        </Badge>
                      </TableCell>
                      <TableCell className="overflow-hidden whitespace-nowrap">
                        <span className="block truncate text-xs font-medium text-slate-700">
                          {item.jenisKelamin === "LAKI_LAKI"
                            ? "Laki-laki"
                            : "Perempuan"}
                        </span>
                      </TableCell>
                      <TableCell className="whitespace-nowrap">
                        <span
                          className="block max-w-[140px] truncate text-xs font-medium text-slate-600"
                          title={item.noHp || "-"}
                        >
                          {item.noHp || "-"}
                        </span>
                      </TableCell>
                      <TableCell className="whitespace-nowrap">
                        <div className="flex items-center gap-1.5">
                          <UserIcon size={12} className="text-slate-400" />
                          <span
                            className="max-w-[120px] truncate text-xs text-slate-600"
                            title={
                              item.user?.name
                                ? capitalizeName(item.user.name)
                                : "-"
                            }
                          >
                            {item.user?.name
                              ? capitalizeName(item.user.name)
                              : "-"}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-2">
                          <Button
                            size="sm"
                            variant="outline"
                            className="h-11 w-11 p-0 border-slate-200 text-slate-500 hover:text-slate-900 transition-colors"
                            asChild
                            title={`Lihat detail ${item.namaLengkap}`}
                          >
                            <Link
                              href={`/dashboard/anggota/${item.id}`}
                              aria-label={`Lihat detail ${item.namaLengkap}`}
                            >
                              <Eye className="w-4 h-4" />
                            </Link>
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            className="h-11 w-11 p-0 border-slate-200 text-slate-500 hover:text-slate-900 transition-colors"
                            asChild
                            title={`Edit ${item.namaLengkap}`}
                          >
                            <Link
                              href={`/dashboard/anggota/${item.id}/edit`}
                              aria-label={`Edit ${item.namaLengkap}`}
                            >
                              <Pencil className="w-4 h-4" />
                            </Link>
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            className="h-11 w-11 p-0 border-slate-200 text-red-500 hover:text-red-600 hover:bg-red-50 transition-colors"
                            onClick={() => setConfirmDeleteId(item.id)}
                            title={`Hapus ${item.namaLengkap}`}
                            aria-label={`Hapus ${item.namaLengkap}`}
                          >
                            <Trash2 className="w-4 h-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))
                )}

                {totalPages >= 1 && (
                  <TableRow className="hover:bg-transparent border-t bg-slate-50/30">
                    <TableCell colSpan={7} className="p-0">
                      <div className="flex items-center justify-between px-4 py-2">
                        <p className="text-xs text-muted-foreground hidden sm:block">
                          Menampilkan{" "}
                          <span className="font-medium text-slate-700">
                            {totalItems === 0 ? 0 : (currentPage - 1) * 10 + 1}
                          </span>{" "}
                          sampai{" "}
                          <span className="font-medium text-slate-700">
                            {Math.min(currentPage * 10, totalItems)}
                          </span>{" "}
                          dari{" "}
                          <span className="font-medium text-slate-700">
                            {totalItems}
                          </span>{" "}
                          anggota
                        </p>
                        <Pagination className="mx-0 w-auto scale-90 sm:scale-100 origin-right">
                          <PaginationContent>
                            <PaginationItem>
                              <PaginationPrevious
                                href="#"
                                aria-disabled={currentPage === 1}
                                tabIndex={currentPage === 1 ? -1 : undefined}
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
                                tabIndex={
                                  currentPage === totalPages ? -1 : undefined
                                }
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
        title="Hapus Data Anggota?"
        description={`Apakah Anda yakin ingin menghapus data anggota "${(() => {
          const name =
            sortedData.find((item) => item.id === confirmDeleteId)
              ?.namaLengkap || "";
          return name.length > 60 ? `${name.slice(0, 57)}...` : name;
        })()}"? Tindakan ini tidak dapat dibatalkan.`}
        variant="destructive"
        loading={isDeleting}
      />
    </div>
  );
}
