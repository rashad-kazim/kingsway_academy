"use client";

import type { RefObject } from "react";
import { ImagePlus, X } from "lucide-react";
import { cn } from "@/lib/utils";
import { ObjectCoverImage } from "./object-cover-image";

type PhotoUploadAvatarProps = {
  inputRef: RefObject<HTMLInputElement | null>;
  name: string;
  onChange: (file: File | null) => void;
  onClear: () => void;
  previewURL: string;
  removeLabel: string;
  removeRingClassName?: string;
};

export function PhotoUploadAvatar({
  inputRef,
  name,
  onChange,
  onClear,
  previewURL,
  removeLabel,
  removeRingClassName,
}: PhotoUploadAvatarProps) {
  return (
    <div className="relative size-36">
      <label className="group relative flex size-36 cursor-pointer items-center justify-center overflow-hidden rounded-full border border-dashed border-kw-c-b9c5d6 bg-kw-c-f7f8fb text-kw-c-0a284b shadow-inner transition hover:border-kw-c-ef2334 hover:bg-kw-c-f1f4f8 dark:border-kw-c-3a4658 dark:bg-kw-c-202b3a dark:text-kw-c-f3f6fa dark:hover:border-kw-c-ff3b4f dark:hover:bg-kw-c-263448">
        {previewURL ? (
          <ObjectCoverImage src={previewURL} />
        ) : (
          <ImagePlus className="size-8 transition-transform group-hover:scale-110" />
        )}
        <input
          ref={inputRef}
          accept="image/jpeg,image/png,image/webp,.jpg,.jpeg,.png,.webp"
          className="sr-only"
          name={name}
          onChange={(event) =>
            onChange(event.currentTarget.files?.[0] ?? null)
          }
          type="file"
        />
      </label>
      {previewURL ? (
        <button
          aria-label={removeLabel}
          className={cn(
            "absolute right-0 top-0 grid size-8 cursor-pointer place-items-center rounded-full bg-kw-c-ef2334 text-white shadow-lg ring-4 ring-white transition hover:bg-kw-c-d91f30 dark:bg-kw-c-ff3b4f dark:hover:bg-kw-c-ff5a69",
            removeRingClassName ?? "dark:ring-kw-c-1b2635",
          )}
          type="button"
          onClick={onClear}
        >
          <X className="size-4" />
        </button>
      ) : null}
    </div>
  );
}
