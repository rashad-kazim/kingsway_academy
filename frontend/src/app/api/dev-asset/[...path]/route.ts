import { readFile, stat } from "fs/promises";
import path from "path";
import { NextResponse } from "next/server";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

const contentTypes: Record<string, string> = {
  ".jpg": "image/jpeg",
  ".jpeg": "image/jpeg",
  ".png": "image/png",
  ".svg": "image/svg+xml",
  ".webp": "image/webp",
};

type DevAssetContext = {
  params: Promise<{
    path: string[];
  }>;
};

export async function GET(_request: Request, context: DevAssetContext) {
  if (process.env.NODE_ENV !== "development") {
    return new NextResponse(null, { status: 404 });
  }

  const { path: segments } = await context.params;
  const publicRoot = path.resolve(process.cwd(), "public");
  const requestedPath = path.resolve(publicRoot, ...segments);

  if (!requestedPath.startsWith(`${publicRoot}${path.sep}`)) {
    return new NextResponse(null, { status: 404 });
  }

  try {
    const [file, info] = await Promise.all([readFile(requestedPath), stat(requestedPath)]);
    const contentType = contentTypes[path.extname(requestedPath).toLowerCase()] ?? "application/octet-stream";

    return new NextResponse(new Uint8Array(file), {
      headers: {
        "Cache-Control": "no-store, no-cache, max-age=0, must-revalidate",
        "Content-Length": String(info.size),
        "Content-Type": contentType,
        ETag: `"${info.mtimeMs}-${info.size}"`,
      },
    });
  } catch {
    return new NextResponse(null, { status: 404 });
  }
}
