"use client";

import { Skeleton } from "@/components/ui/skeleton";
import { Card, CardContent } from "@/components/ui/card";

export function PeriodeSkeleton() {
  return (
    <div className="flex flex-col gap-4 sm:gap-6 animate-pulse">
      {/* Header Skeleton */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <Skeleton className="h-10 w-10 rounded-lg" />
          <div className="space-y-2">
            <Skeleton className="h-6 w-48 md:w-64" />
            <Skeleton className="h-4 w-64 md:w-80" />
          </div>
        </div>
        <Skeleton className="h-10 w-full sm:w-40 rounded-md" />
      </div>

      {/* List Card Skeleton */}
      <div className="grid gap-4">
        {[...Array(4)].map((_, i) => (
          <div
            key={i}
            className="flex flex-col gap-3 p-4 sm:p-5 rounded-xl border border-slate-200 bg-card shadow-sm sm:flex-row sm:items-center sm:justify-between sm:gap-4"
          >
            <div className="flex items-start justify-between sm:block gap-2 min-w-0">
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <Skeleton className="h-6 w-32 rounded-md" />
                  <Skeleton className="h-5 w-16 rounded-full" />
                </div>
                <Skeleton className="h-3.5 w-36" />
              </div>
              <div className="flex items-center gap-1 sm:hidden">
                <Skeleton className="h-8 w-16 rounded-md" />
              </div>
            </div>

            <div className="flex items-center gap-2 w-full pt-2.5 sm:pt-0 border-t sm:border-t-0 border-slate-100 sm:w-auto">
              <Skeleton className="h-9 flex-1 sm:flex-initial sm:w-36 rounded-md" />
              <div className="hidden sm:flex sm:items-center sm:gap-2">
                <Skeleton className="h-9 w-20 rounded-md" />
                <Skeleton className="h-9 w-9 rounded-md" />
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

export function PeriodeFormSkeleton() {
  return (
    <div className="max-w-xl flex flex-col gap-4 sm:gap-6 animate-pulse">
      {/* Header Skeleton */}
      <div className="flex items-center gap-4">
        <Skeleton className="h-10 w-10 rounded-md" />
        <div className="space-y-2">
          <Skeleton className="h-6 w-48" />
          <Skeleton className="h-4 w-64" />
        </div>
      </div>

      <Card className="shadow-none border-slate-200">
        <CardContent className="p-6 space-y-6">
          <div className="space-y-4">
            <div className="space-y-2">
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-10 w-full rounded-md" />
            </div>
          </div>

          <div className="flex gap-3 mt-6 pt-6 border-t">
            <Skeleton className="h-11 flex-1 rounded-md" />
            <Skeleton className="h-11 flex-1 rounded-md" />
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
