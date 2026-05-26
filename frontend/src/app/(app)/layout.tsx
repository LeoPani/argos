import { Sidebar } from "@/components/layout/Sidebar";
import { CommandPalette } from "@/components/ui/command-palette";

export default function AppLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <>
      <Sidebar />
      <main className="flex-1 ml-56 min-h-screen overflow-y-auto">
        {children}
      </main>
      <CommandPalette />
    </>
  );
}
