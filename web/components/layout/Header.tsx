import { Database } from "lucide-react";

export function Header() {
  return (
    <header className="sticky top-0 z-50 w-full border-b border-zinc-800 bg-zinc-950/80 backdrop-blur supports-[backdrop-filter]:bg-zinc-950/60">
      <div className="container flex h-14 items-center">
        <div className="flex items-center gap-2 px-4 md:px-8 font-semibold">
          <Database className="h-5 w-5" />
          <span>KVStore Admin</span>
        </div>
      </div>
    </header>
  );
}
