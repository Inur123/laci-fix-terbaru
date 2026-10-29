"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { cn } from "@/lib/utils";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Checkbox } from "@/components/ui/checkbox";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Search, UserPlus, Loader2, Copy } from "lucide-react";
import { Input } from "@/components/ui/input";
import { getPeriodes } from "@/app/actions/periode-actions";
import {
  getAnggotaList,
  copyAnggotaToCurrentPeriode,
} from "@/app/actions/anggota-actions";
import { toast } from "sonner";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Label } from "@/components/ui/label";
import { fetchAllPages } from "@/lib/fetch-all-pages";

type CopyMember = {
  id: string;
  namaLengkap: string;
  jabatan?: string | null;
};

export function CopyMemberDialog({ userRole }: { userRole: string }) {
  const isCabang = userRole === "SEKRETARIS_CABANG";
  const router = useRouter();

  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [fetchingMembers, setFetchingMembers] = useState(false);

  const [periodes, setPeriodes] = useState<
    { id: string; nama: string; isActive: boolean }[]
  >([]);
  const [selectedPeriodId, setSelectedPeriodId] = useState<string>("");

  const [members, setMembers] = useState<CopyMember[]>([]);
  const [selectedMemberIds, setSelectedMemberIds] = useState<string[]>([]);
  const [search, setSearch] = useState("");

  useEffect(() => {
    if (open) {
      const loadPeriodes = async () => {
        try {
          const data = await getPeriodes();
          setPeriodes(
            data.filter(
              (periode: { isActive: boolean }) => !periode.isActive,
            ) as { id: string; nama: string; isActive: boolean }[],
          );
        } catch {
          toast.error("Gagal memuat daftar periode.");
        }
      };
      loadPeriodes();
    }
  }, [open]);

  useEffect(() => {
    if (selectedPeriodId) {
      const loadMembers = async () => {
        setFetchingMembers(true);
        try {
          const data = await fetchAllPages<CopyMember>((page, limit) =>
            getAnggotaList("", page, limit, undefined, selectedPeriodId),
          );
          setMembers(data);
          setSelectedMemberIds([]);
        } catch (error) {
          toast.error("Gagal memuat data anggota");
        } finally {
          setFetchingMembers(false);
        }
      };
      loadMembers();
    } else {
      setMembers([]);
    }
  }, [selectedPeriodId]);

  const filteredMembers = members.filter(
    (m) =>
      m.namaLengkap.toLowerCase().includes(search.toLowerCase()) ||
      (m.jabatan && m.jabatan.toLowerCase().includes(search.toLowerCase())),
  );
  const allFilteredSelected =
    filteredMembers.length > 0 &&
    filteredMembers.every((member) => selectedMemberIds.includes(member.id));

  const toggleMember = (id: string) => {
    setSelectedMemberIds((prev) =>
      prev.includes(id) ? prev.filter((mid) => mid !== id) : [...prev, id],
    );
  };

  const toggleAll = () => {
    if (allFilteredSelected) {
      const visibleIds = new Set(filteredMembers.map((member) => member.id));
      setSelectedMemberIds((previous) =>
        previous.filter((id) => !visibleIds.has(id)),
      );
    } else {
      setSelectedMemberIds((previous) => [
        ...new Set([
          ...previous,
          ...filteredMembers.map((member) => member.id),
        ]),
      ]);
    }
  };

  const handleCopy = async () => {
    if (selectedMemberIds.length === 0) {
      toast.error("Pilih minimal satu anggota");
      return;
    }

    setLoading(true);
    try {
      const result = await copyAnggotaToCurrentPeriode(selectedMemberIds);
      if (result.success) {
        toast.success(result.success);
        setOpen(false);
        setSelectedPeriodId("");
        setSelectedMemberIds([]);
        setSearch("");
        router.refresh();
      } else {
        toast.error(result.error);
      }
    } catch (error) {
      toast.error("Gagal menyalin anggota");
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        setOpen(nextOpen);
        if (!nextOpen && !loading) {
          setSelectedPeriodId("");
          setSelectedMemberIds([]);
          setSearch("");
          setMembers([]);
        }
      }}
    >
      <DialogTrigger asChild>
        <Button
          variant="outline"
          className={cn(
            "min-h-11 w-full sm:w-auto px-4 text-sm bg-white shadow-sm transition-all duration-200",
            isCabang
              ? "border-blue-200 text-blue-600 hover:bg-blue-50 hover:text-blue-700"
              : "border-green-200 text-green-600 hover:bg-green-50 hover:text-green-700",
          )}
        >
          <Copy className="mr-2 h-4 w-4" />
          Salin dari Periode Lain
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-[500px] gap-0 p-0 overflow-hidden">
        <DialogHeader className="p-6 pb-4 bg-slate-50/50 border-b">
          <DialogTitle className="flex items-center gap-2">
            <UserPlus
              className={cn(
                "h-5 w-5",
                isCabang ? "text-blue-600" : "text-green-600",
              )}
            />
            Salin Anggota
          </DialogTitle>
          <DialogDescription>
            Pilih periode asal dan centang anggota yang akan dilanjutkan ke
            periode saat ini.
          </DialogDescription>
        </DialogHeader>

        <div className="p-6 space-y-4">
          <div className="space-y-2">
            <Label
              htmlFor="copy-member-period"
              className="text-xs font-semibold text-slate-500 uppercase tracking-wider"
            >
              Pilih Periode Asal
            </Label>
            <Select
              value={selectedPeriodId}
              onValueChange={setSelectedPeriodId}
            >
              <SelectTrigger
                id="copy-member-period"
                aria-label="Pilih periode asal anggota"
                className="min-h-11 w-full bg-white"
              >
                <SelectValue placeholder="Pilih periode..." />
              </SelectTrigger>
              <SelectContent>
                {periodes.length === 0 ? (
                  <div className="p-4 text-center text-sm text-slate-500">
                    Tidak ada periode lain
                  </div>
                ) : (
                  periodes.map((p) => (
                    <SelectItem key={p.id} value={p.id}>
                      {p.nama} {p.isActive && "(Aktif)"}
                    </SelectItem>
                  ))
                )}
              </SelectContent>
            </Select>
          </div>

          {selectedPeriodId && (
            <div className="space-y-3 animate-in fade-in slide-in-from-top-2 duration-300">
              <div className="relative">
                <Label htmlFor="copy-member-search" className="sr-only">
                  Cari anggota periode asal
                </Label>
                <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-slate-400" />
                <Input
                  id="copy-member-search"
                  placeholder="Cari nama atau jabatan..."
                  className="min-h-11 pl-9"
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                />
              </div>

              <div className="flex items-center justify-between px-1">
                <div className="text-xs font-medium text-slate-500">
                  {selectedMemberIds.length} terpilih dari{" "}
                  {filteredMembers.length}
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  className={cn(
                    "min-h-11 px-3 text-xs font-bold",
                    isCabang
                      ? "text-blue-600 hover:text-blue-700 hover:bg-blue-50"
                      : "text-green-600 hover:text-green-700 hover:bg-green-50",
                  )}
                  onClick={toggleAll}
                  disabled={filteredMembers.length === 0}
                >
                  {allFilteredSelected ? "Batal Semua" : "Pilih Semua"}
                </Button>
              </div>

              <ScrollArea className="h-[250px] rounded-md border bg-white p-1">
                {fetchingMembers ? (
                  <div className="flex flex-col items-center justify-center h-[240px] gap-2 text-slate-400">
                    <Loader2 className="h-6 w-6 animate-spin" />
                    <p className="text-xs">Memuat anggota...</p>
                  </div>
                ) : filteredMembers.length === 0 ? (
                  <div className="flex flex-col items-center justify-center h-[240px] text-slate-400">
                    <p className="text-xs">Tidak ada anggota ditemukan</p>
                  </div>
                ) : (
                  <div className="space-y-1">
                    {filteredMembers.map((m) => (
                      <label
                        key={m.id}
                        htmlFor={`copy-member-${m.id}`}
                        className={cn(
                          "flex min-h-11 items-center gap-3 p-2 rounded-md transition-colors cursor-pointer hover:bg-slate-50",
                          selectedMemberIds.includes(m.id) &&
                            (isCabang ? "bg-blue-50/50" : "bg-green-50/50"),
                        )}
                      >
                        <Checkbox
                          id={`copy-member-${m.id}`}
                          checked={selectedMemberIds.includes(m.id)}
                          onCheckedChange={() => toggleMember(m.id)}
                          aria-label={`Pilih ${m.namaLengkap}`}
                        />
                        <div className="flex items-center gap-3 min-w-0 flex-1">
                          <Avatar className="h-8 w-8 border border-slate-100">
                            <AvatarFallback
                              className={cn(
                                "text-[10px] font-bold",
                                isCabang
                                  ? "bg-blue-50 text-blue-600"
                                  : "bg-green-50 text-green-600",
                              )}
                            >
                              {m.namaLengkap.slice(0, 2).toUpperCase()}
                            </AvatarFallback>
                          </Avatar>
                          <div className="flex flex-col min-w-0">
                            <span className="text-sm font-semibold text-slate-900 truncate">
                              {m.namaLengkap}
                            </span>
                            {m.jabatan && (
                              <span className="text-[10px] text-slate-500 truncate">
                                {m.jabatan}
                              </span>
                            )}
                          </div>
                        </div>
                      </label>
                    ))}
                  </div>
                )}
              </ScrollArea>
            </div>
          )}
        </div>

        <DialogFooter className="p-6 bg-slate-50/50 border-t">
          <Button
            variant="ghost"
            onClick={() => setOpen(false)}
            disabled={loading}
            className="min-h-11"
          >
            Batal
          </Button>
          <Button
            onClick={handleCopy}
            disabled={loading || selectedMemberIds.length === 0}
            className={cn(
              "min-h-11 text-white min-w-[100px] shadow-md",
              isCabang
                ? "bg-blue-600 hover:bg-blue-700 shadow-blue-200"
                : "bg-green-600 hover:bg-green-700 shadow-green-200",
            )}
          >
            {loading ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Menyalin...
              </>
            ) : (
              `Salin ${selectedMemberIds.length} Anggota`
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
