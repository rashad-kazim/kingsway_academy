import Image from "next/image";

type ObjectCoverImageProps = {
  alt?: string;
  className?: string;
  src: string;
};

export function ObjectCoverImage({
  alt = "",
  className = "size-full object-cover",
  src,
}: ObjectCoverImageProps) {
  return (
    <Image
      alt={alt}
      className={className}
      fill
      sizes="144px"
      src={src}
      unoptimized
    />
  );
}
