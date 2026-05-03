export function appAsset(path: string) {
  const normalized = path.startsWith("/") ? path : `/${path}`;

  if (process.env.NODE_ENV === "development") {
    return `/api/dev-asset${normalized}`;
  }

  return normalized;
}
