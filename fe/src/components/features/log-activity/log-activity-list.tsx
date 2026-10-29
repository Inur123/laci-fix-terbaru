"use client";

import React, { useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { cn } from "@/lib/utils";
import { UserFilterSelect } from "@/components/shared/user-filter-select";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import {
  RefreshCcw,
  Eye,
  ArrowUpDown,
  ArrowUp,
  ArrowDown,
  Search,
} from "lucide-react";
import Link from "next/link";
import type { LogAction, LogModule } from "@/types/domain";
import { DatePickerWithPresets } from "@/components/ui/date-range-picker-presets";
import { DateRange } from "react-day-picker";
import {
  getPersonalLogs,
  getGlobalLogs,
  type LogActivityFilters,
} from "@/app/actions/log-activity-actions";
import { formatDateForInput } from "@/lib/date-utils";
import { toast } from "sonner";

export type LogActivityData = {
  id: string;
  action: string;
  module: LogModule;
  description: string;
  createdAt: Date;
  user: {
    id: string;
    name: string;
    email: string;
    role: string;
  };
  periode: {
    id: string;
    nama: string;
  };
};

const actionConfig: Record<LogAction, { label: string; className: string }> = {
  CREATE: {
    label: "Tambah",
    className:
      "bg-green-100/80 text-green-700 border-green-200 hover:bg-green-200/80",
  },
  UPDATE: {
    label: "Update",
    className:
      "bg-blue-100/80 text-blue-700 border-blue-200 hover:bg-blue-200/80",
  },
  DELETE: {
    label: "Hapus",
    className: "bg-red-100/80 text-red-700 border-red-200 hover:bg-red-200/80",
  },
  EXPORT: {
    label: "Export Excel",
    className:
      "bg-purple-100/80 text-purple-700 border-purple-200 hover:bg-purple-200/80",
  },
  APPROVE: {
    label: "Update",
    className:
      "bg-blue-100/80 text-blue-700 border-blue-200 hover:bg-blue-200/80",
  },
  REJECT: {
    label: "Update",
    className:
      "bg-blue-100/80 text-blue-700 border-blue-200 hover:bg-blue-100/80",
  },
  LOGIN: {
    label: "Login",
    className:
      "bg-emerald-100/80 text-emerald-700 border-emerald-200 hover:bg-emerald-200/80",
  },
  LOGOUT: {
    label: "Logout",
    className:
      "bg-orange-100/80 text-orange-700 border-orange-200 hover:bg-orange-200/80",
  },
};

const getActionInfo = (action: string) =>
  actionConfig[action as LogAction] ?? {
    label: "Aktivitas lama",
    className: "bg-slate-100 text-slate-700 border-slate-200",
  };

const moduleConfig: Record<LogModule, { label: string; className: string }> = {
  ARSIP_SURAT: {
    label: "Arsip Surat",
    className: "bg-blue-50 text-blue-600 border-blue-200 hover:bg-blue-100",
  },
  ANGGOTA: {
    label: "Anggota",
    className: "bg-green-50 text-green-600 border-green-200 hover:bg-green-100",
  },
  BERKAS_PIMPINAN: {
    label: "Berkas Pimpinan",
    className:
      "bg-purple-50 text-purple-600 border-purple-200 hover:bg-purple-100",
  },
  BERKAS_SP: {
    label: "Berkas SP",
    className:
      "bg-indigo-50 text-indigo-600 border-indigo-200 hover:bg-indigo-100",
  },
  AGENDA_KEGIATAN: {
    label: "Kegiatan",
    className: "bg-amber-50 text-amber-600 border-amber-200 hover:bg-amber-100",
  },
  PENGAJUAN_BERKAS: {
    label: "Pengajuan PAC",
    className: "bg-rose-50 text-rose-600 border-rose-200 hover:bg-rose-100",
  },
  PERIODE: {
    label: "Periode",
    className: "bg-cyan-50 text-cyan-600 border-cyan-200 hover:bg-cyan-100",
  },
  USER: {
    label: "Update Profil",
    className: "bg-slate-50 text-slate-600 border-slate-200 hover:bg-slate-100",
  },
  AUTH: {
    label: "Autentikasi",
    className: "bg-gray-50 text-gray-600 border-gray-200 hover:bg-gray-100",
  },
  PRESENSI: {
    label: "Presensi",
    className:
      "bg-emerald-50 text-emerald-600 border-emerald-200 hover:bg-emerald-100",
  },
};

type LogActivityListProps = {
  initialLogs: LogActivityData[];
  initialTotalPages: number;
  initialCurrentPage: number;
  initialTotalItems: number;
  initialView: "personal" | "global";
  userRole: string;
  pacUsers?: { id: string; name: string }[];
};

export function LogActivityList({
  initialLogs,
  initialTotalPages,
  initialCurrentPage,
  initialTotalItems,
  initialView,
  userRole,
  pacUsers = [],
}: LogActivityListProps) {
  const [logs, setLogs] = useState<LogActivityData[]>(initialLogs);
  const [totalPages, setTotalPages] = useState(initialTotalPages);
  const [currentPage, setCurrentPage] = useState(initialCurrentPage);
  const [totalItems, setTotalItems] = useState(initialTotalItems);
  const currentView = initialView;
  const [isLoading, setIsLoading] = useState(false);
  const realtimeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  React.useEffect(() => {
    setLogs(initialLogs);
    setTotalPages(initialTotalPages);
    setTotalItems(initialTotalItems);
    setCurrentPage(initialCurrentPage);
  }, [initialLogs, initialTotalPages, initialTotalItems, initialCurrentPage]);

  const [searchTerm, setSearchTerm] = useState("");
  const [actionFilter, setActionFilter] = useState("ALL");
  const [moduleFilter, setModuleFilter] = useState("ALL");
  const [dateRange, setDateRange] = useState<DateRange | undefined>(undefined);
  const [userFilter, setUserFilter] = useState("ALL");

  const isCabang = userRole === "SEKRETARIS_CABANG";

  type LogSortKey = "createdAt" | "action" | "module" | "userName";
  type SortDir = "asc" | "desc";
  const [sortKey, setSortKey] = useState<LogSortKey | null>("createdAt");
  const [sortDir, setSortDir] = useState<SortDir>("desc");

  const handleSort = (key: LogSortKey) => {
    const nextDir: SortDir =
      sortKey === key && sortDir === "asc" ? "desc" : "asc";
    setSortKey(key);
    setSortDir(nextDir);
    setCurrentPage(1);
  };

  const LogSortIcon = ({ col }: { col: LogSortKey }) => {
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

  const sortedLogs = logs;

  const isDirty =
    currentPage !== 1 ||
    searchTerm !== "" ||
    actionFilter !== "ALL" ||
    moduleFilter !== "ALL" ||
    dateRange !== undefined ||
    userFilter !== "ALL" ||
    sortKey !== "createdAt" ||
    sortDir !== "desc";

  const fetchData = React.useCallback(
    async (
      search: string,
      action: string,
      module: string,
      start: string,
      end: string,
      page: number,
      view: "personal" | "global",
      userId: string,
      sKey: LogSortKey | null,
      sDir: SortDir,
      options: { silent?: boolean } = {},
    ) => {
      if (!options.silent) setIsLoading(true);
      try {
        const filters: LogActivityFilters = {};
        if (search) filters.search = search;
        if (action !== "ALL") filters.action = action;
        if (module !== "ALL") filters.module = module;
        if (start) filters.startDate = start;
        if (end) filters.endDate = end;
        if (userId && userId !== "ALL") filters.userId = userId;
        if (sKey) filters.sortKey = sKey;
        if (sDir) filters.sortDir = sDir;

        const data =
          view === "global"
            ? await getGlobalLogs(filters, page, 20)
            : await getPersonalLogs(filters, page, 20);

        setLogs(data.data as LogActivityData[]);
        setTotalPages(data.totalPages);
        setTotalItems(data.total);
      } catch (error) {
        console.error("Error fetching logs:", error);
        if (!options.silent) toast.error("Gagal memuat riwayat aktivitas.");
      } finally {
        if (!options.silent) setIsLoading(false);
      }
    },
    [],
  );

  React.useEffect(() => {
    const handler = (event: Event) => {
      const detail = (event as CustomEvent).detail as {
        type?: string;
        model?: string;
      };
      const isLogEvent =
        detail?.type === "log" ||
        (detail?.type === "mutation" && detail.model === "LogActivity");
      if (!isLogEvent) return;
      if (realtimeTimerRef.current) return;
      realtimeTimerRef.current = setTimeout(() => {
        realtimeTimerRef.current = null;
        const start = formatDateForInput(dateRange?.from);
        const end = formatDateForInput(dateRange?.to);
        fetchData(
          searchTerm,
          actionFilter,
          moduleFilter,
          start,
          end,
          currentPage,
          currentView,
          userFilter,
          sortKey,
          sortDir,
          { silent: true },
        );
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
  }, [
    searchTerm,
    actionFilter,
    moduleFilter,
    dateRange,
    currentPage,
    currentView,
    userFilter,
    sortKey,
    sortDir,
    fetchData,
  ]);

  const isFirstRender = useRef(true);

  React.useEffect(() => {
    if (isFirstRender.current) {
      isFirstRender.current = false;
      return;
    }
    const timer = setTimeout(() => {
      const start = formatDateForInput(dateRange?.from);
      const end = formatDateForInput(dateRange?.to || dateRange?.from);

      fetchData(
        searchTerm,
        actionFilter,
        moduleFilter,
        start,
        end,
        currentPage,
        currentView,
        userFilter,
        sortKey,
        sortDir,
      );
    }, 350);

    return () => clearTimeout(timer);
  }, [
    searchTerm,
    actionFilter,
    moduleFilter,
    dateRange,
    currentPage,
    currentView,
    userFilter,
    sortKey,
    sortDir,
    fetchData,
  ]);

  const handlePageChange = (page: number) => {
    setCurrentPage(page);
  };

  const handleActionFilterChange = (val: string) => {
    setActionFilter(val);
    setCurrentPage(1);
  };

  const handleModuleFilterChange = (val: string) => {
    setModuleFilter(val);
    setCurrentPage(1);
  };

  const handleUserFilterSelect = (val: string) => {
    setUserFilter(val);
    setCurrentPage(1);
  };

  const handleReset = () => {
    setSearchTerm("");
    setActionFilter("ALL");
    setModuleFilter("ALL");
    setDateRange(undefined);
    setUserFilter("ALL");
    setSortKey("createdAt");
    setSortDir("desc");
    setCurrentPage(1);
  };


  return (
    <div className="space-y-6">
      <div
        className={cn(
          "mb-4 grid grid-cols-1 items-end gap-4 sm:grid-cols-2 lg:grid-cols-3",
          currentView === "global" && isCabang
            ? "xl:grid-cols-[1.15fr_1.15fr_1fr_1fr_1fr_auto]"
            : "xl:grid-cols-[1.15fr_1.15fr_1fr_1fr_auto]",
        )}
      >
        <div className="relative w-full">
          <Label htmlFor="date" className="mb-1 block text-xs font-medium">
            Rentang Tanggal
          </Label>
          <DatePickerWithPresets
            date={dateRange}
            onSelect={(value) => {
              setDateRange(value);
              setCurrentPage(1);
            }}
            className="w-full bg-white border-slate-200 h-9"
          />
        </div>

        {currentView === "global" && isCabang && pacUsers.length > 0 && (
          <div className="w-full">
            <Label className="text-xs font-medium mb-1 block">
              Filter User
            </Label>
            <UserFilterSelect
              users={pacUsers}
              selectedUserId={userFilter}
              onSelectUser={handleUserFilterSelect}
              placeholder="Pilih User"
            />
          </div>
        )}

        <div className="w-full">
          <Label htmlFor="activity-search" className="mb-1 block text-xs font-medium">
            Cari Aktivitas
          </Label>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
            <Input
              id="activity-search"
              value={searchTerm}
              onChange={(event) => {
                setSearchTerm(event.target.value);
                setCurrentPage(1);
              }}
              placeholder="Deskripsi aktivitas..."
              className="h-9 bg-white pl-9"
            />
          </div>
        </div>

        <div className="w-full">
          <Label htmlFor="activity-action" className="mb-1 block text-xs font-medium">
            Entitas
          </Label>
          <Select value={actionFilter} onValueChange={handleActionFilterChange}>
            <SelectTrigger id="activity-action" className="w-full bg-white h-9 text-sm border-slate-200 shadow-sm">
              <SelectValue placeholder="Semua" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="ALL">Semua</SelectItem>
              {Object.entries(actionConfig)
                .filter(([key]) => !["APPROVE", "REJECT"].includes(key))
                .map(([key, config]) => (
                  <SelectItem key={key} value={key}>
                    {config.label}
                  </SelectItem>
                ))}
            </SelectContent>
          </Select>
        </div>

        <div className="w-full">
          <Label htmlFor="activity-module" className="mb-1 block text-xs font-medium">
            Modul/Menu
          </Label>
          <Select value={moduleFilter} onValueChange={handleModuleFilterChange}>
            <SelectTrigger id="activity-module" className="w-full bg-white h-9 text-sm border-slate-200 shadow-sm">
              <SelectValue placeholder="Semua" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="ALL">Semua</SelectItem>
              {Object.entries(moduleConfig).map(([key, config]) => (
                <SelectItem key={key} value={key}>
                  {config.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="w-full md:w-auto">
          <Button
            variant="outline"
            className={cn(
              "h-9 w-full md:w-auto px-4 text-sm bg-white border-slate-200 shadow-sm whitespace-nowrap transition-all duration-200",
              isDirty
                ? "text-slate-900 border-slate-300 opacity-100"
                : "text-slate-400 border-slate-200 opacity-50 cursor-not-allowed",
            )}
            onClick={handleReset}
            disabled={!isDirty || isLoading}
          >
            <RefreshCcw className="mr-2 h-3.5 w-3.5" />
            Reset
          </Button>
        </div>
      </div>

      <div className="relative" aria-busy={isLoading}>
        <div className="rounded-md border">
          <div className="overflow-x-auto">
            <Table className="w-full [&_td]:py-3 [&_th]:py-3">
              <TableHeader className="sticky top-0 bg-white z-10 shadow-[0_1px_0_0_rgba(0,0,0,0.05)]">
                <TableRow className="bg-slate-50/40 hover:bg-slate-50/40 border-b-slate-100">
                  <TableHead className="w-[50px] text-center text-slate-500 font-semibold h-12 whitespace-nowrap">
                    No
                  </TableHead>
                  <TableHead
                    className="w-[140px] whitespace-nowrap text-slate-500 font-semibold h-12 cursor-pointer select-none hover:bg-slate-100 transition-colors"
                    aria-sort={
                      sortKey === "createdAt"
                        ? sortDir === "asc"
                          ? "ascending"
                          : "descending"
                        : "none"
                    }
                  >
                    <button
                      type="button"
                      className="inline-flex items-center rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      onClick={() => handleSort("createdAt")}
                    >
                      Waktu
                      <LogSortIcon col="createdAt" />
                    </button>
                  </TableHead>
                  {currentView === "global" && (
                    <TableHead
                      className="w-[130px] whitespace-nowrap text-slate-500 font-semibold h-12 cursor-pointer select-none hover:bg-slate-100 transition-colors"
                      aria-sort={
                        sortKey === "userName"
                          ? sortDir === "asc"
                            ? "ascending"
                            : "descending"
                          : "none"
                      }
                    >
                      <button
                        type="button"
                        className="inline-flex items-center rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                        onClick={() => handleSort("userName")}
                      >
                        User
                        <LogSortIcon col="userName" />
                      </button>
                    </TableHead>
                  )}
                  <TableHead
                    className="w-[100px] whitespace-nowrap text-slate-500 font-semibold h-12 cursor-pointer select-none hover:bg-slate-100 transition-colors"
                    aria-sort={
                      sortKey === "action"
                        ? sortDir === "asc"
                          ? "ascending"
                          : "descending"
                        : "none"
                    }
                  >
                    <button
                      type="button"
                      className="inline-flex items-center rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      onClick={() => handleSort("action")}
                    >
                      Entitas
                      <LogSortIcon col="action" />
                    </button>
                  </TableHead>
                  <TableHead
                    className="w-[110px] whitespace-nowrap text-slate-500 font-semibold h-12 cursor-pointer select-none hover:bg-slate-100 transition-colors"
                    aria-sort={
                      sortKey === "module"
                        ? sortDir === "asc"
                          ? "ascending"
                          : "descending"
                        : "none"
                    }
                  >
                    <button
                      type="button"
                      className="inline-flex items-center rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      onClick={() => handleSort("module")}
                    >
                      Menu
                      <LogSortIcon col="module" />
                    </button>
                  </TableHead>
                  <TableHead className="text-slate-500 font-semibold h-12">
                    Aktivitas
                  </TableHead>
                  <TableHead className="w-[60px] text-right text-slate-500 font-semibold h-12">
                    Aksi
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {logs.length === 0 ? (
                  <TableRow>
                    <TableCell
                      colSpan={currentView === "global" ? 7 : 6}
                      className="h-32 text-center text-muted-foreground"
                    >
                      Belum ada riwayat aktivitas pada periode ini.
                    </TableCell>
                  </TableRow>
                ) : (
                  sortedLogs.map((log, index) => {
                    const actionInfo = getActionInfo(log.action);
                    return (
                      <TableRow
                        key={log.id}
                        className="group transition-all hover:bg-slate-50/50"
                      >
                        <TableCell className="text-center text-muted-foreground font-medium whitespace-nowrap">
                          {(currentPage - 1) * 20 + index + 1}
                        </TableCell>
                        <TableCell className="whitespace-nowrap text-xs text-slate-600">
                          {new Date(log.createdAt).toLocaleDateString("id-ID", {
                            day: "numeric",
                            month: "long",
                            year: "numeric",
                          })}
                          {" - "}
                          {new Date(log.createdAt).toLocaleTimeString("id-ID", {
                            hour: "2-digit",
                            minute: "2-digit",
                            second: "2-digit",
                          })}
                        </TableCell>
                        {currentView === "global" && (
                          <TableCell
                            className="max-w-[120px] truncate whitespace-nowrap"
                            title={log.user.name}
                          >
                            {log.user.name}
                          </TableCell>
                        )}
                        <TableCell className="whitespace-nowrap">
                          <Badge
                            variant="outline"
                            className={cn(
                              "whitespace-nowrap transition-colors",
                              actionInfo.className,
                            )}
                          >
                            {actionInfo.label}
                          </Badge>
                        </TableCell>
                        <TableCell className="whitespace-nowrap">
                          <Badge
                            variant="outline"
                            className={cn(
                              "whitespace-nowrap transition-colors",
                              moduleConfig[log.module]?.className,
                            )}
                          >
                            {moduleConfig[log.module]?.label || log.module}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-[13px] leading-relaxed text-slate-600">
                          <div
                            className="max-w-[120px] truncate sm:max-w-[180px] md:max-w-[240px] lg:max-w-[340px] xl:max-w-[480px]"
                            title={log.description}
                          >
                            {log.description}
                          </div>
                        </TableCell>
                        <TableCell className="text-right">
                          <Button
                            size="sm"
                            variant="outline"
                            className="h-11 w-11 border-slate-200 p-0 text-slate-500 transition-colors hover:text-slate-900 sm:h-8 sm:w-8"
                            asChild
                          >
                            <Link
                              href={`/dashboard/log-activity/${log.id}`}
                              aria-label={`Lihat detail aktivitas ${log.description}`}
                            >
                              <Eye className="h-4 w-4" />
                            </Link>
                          </Button>
                        </TableCell>
                      </TableRow>
                    );
                  })
                )}
                {totalPages >= 1 && (
                  <TableRow className="hover:bg-transparent border-t bg-slate-50/30">
                    <TableCell
                      colSpan={currentView === "global" ? 7 : 6}
                      className="p-0"
                    >
                      <div className="flex items-center justify-start sm:justify-between px-4 py-2">
                        <p className="text-xs text-muted-foreground hidden sm:block">
                          Menampilkan{" "}
                          <span className="font-medium text-slate-700">
                            {(currentPage - 1) * 20 + 1}
                          </span>{" "}
                          sampai{" "}
                          <span className="font-medium text-slate-700">
                            {Math.min(currentPage * 20, totalItems)}
                          </span>{" "}
                          dari{" "}
                          <span className="font-medium text-slate-700">
                            {new Intl.NumberFormat("id-ID").format(totalItems)}
                          </span>{" "}
                          data
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
                                aria-disabled={currentPage === 1}
                                tabIndex={currentPage === 1 ? -1 : 0}
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
                                aria-disabled={currentPage === totalPages}
                                tabIndex={currentPage === totalPages ? -1 : 0}
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
    </div>
  );
}
