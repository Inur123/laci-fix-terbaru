"use client";

import { useState, useEffect, useRef } from "react";
import { updatePeriode } from "@/app/actions/periode-actions";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent } from "@/components/ui/card";
import { toast } from "sonner";
import { useRouter } from "next/navigation";
import type { Periode } from "@/types/domain";
import { Spinner } from "@/components/ui/spinner";
import { Upload, X } from "lucide-react";

export function EditPeriodeForm({
  periode,
  userRole,
}: {
  periode: Periode;
  userRole?: string;
}) {
  const [pending, setPending] = useState(false);
  const router = useRouter();

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    const nama = String(formData.get("nama") ?? "").trim();
    if (!nama) {
      toast.error("Nama periode wajib diisi.");
      return;
    }

    setPending(true);
    try {
      const result = await updatePeriode(periode.id, nama);
      if (result.error) {
        toast.error(result.error);
      } else {
        toast.success("Periode berhasil diperbarui!");
        router.push("/dashboard/periode");
        router.refresh();
      }
    } finally {
      setPending(false);
    }
  }

  const formRef = useRef<HTMLFormElement>(null);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement;
      if (e.key === "Enter" && target.tagName !== "TEXTAREA") {
        e.preventDefault();
        formRef.current?.requestSubmit();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  return (
    <form ref={formRef} onSubmit={handleSubmit}>
      <Card>
        <CardContent className="p-6">
          <div className="space-y-2">
            <Label htmlFor="nama">
              Nama Periode <span className="text-red-500">*</span>
            </Label>
            <Input
              id="nama"
              name="nama"
              defaultValue={periode.nama}
              required
              maxLength={100}
              autoComplete="off"
            />
          </div>

          {/* Submit Buttons */}
          <div className="flex gap-3 mt-6 pt-6 border-t">
            <Button
              type="button"
              variant="outline"
              className="flex-1 hover:bg-slate-100 transition-all duration-200"
              onClick={() => router.back()}
              disabled={pending}
            >
              <X className="w-4 h-4 mr-2" />
              Batal
            </Button>
            <Button
              type="submit"
              disabled={pending}
              className={`flex-1 text-white shadow-md hover:shadow-xl transition-all duration-200 ${
                userRole === "SEKRETARIS_CABANG"
                  ? "bg-blue-600 hover:bg-blue-700"
                  : "bg-green-600 hover:bg-green-700"
              }`}
            >
              {pending ? (
                <>
                  <Spinner className="mr-2 h-4 w-4" />
                  Menyimpan...
                </>
              ) : (
                <>
                  <Upload className="w-4 h-4 mr-2" />
                  Simpan
                </>
              )}
            </Button>
          </div>
        </CardContent>
      </Card>
    </form>
  );
}
