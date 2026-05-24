import type { Metadata } from "next";
import "./globals.css";
import { Sidebar } from "@/components/layout/Sidebar";
import { ToastProvider } from "@/components/ui/toast";
import { CommandPalette } from "@/components/ui/command-palette";

export const metadata: Metadata = {
  title: "Argos — IP Intelligence",
  description: "Plataforma de inteligência competitiva para propriedade intelectual",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="pt-BR" className="h-full">
      <body className="h-full flex" style={{ background: "var(--bg)" }}>
        <ToastProvider>
          <Sidebar />
          <main className="flex-1 ml-56 min-h-screen overflow-y-auto">
            {children}
          </main>
          <CommandPalette />
        </ToastProvider>
      </body>
    </html>
  );
}
