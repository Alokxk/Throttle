export function Logo({ className = "h-6 w-6" }: { className?: string }) {
  return <img src="/logo.png" alt="" className={`${className} rounded-md`} />;
}
