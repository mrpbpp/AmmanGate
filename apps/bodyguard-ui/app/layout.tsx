import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";

// Force dynamic rendering for all pages
export const dynamic = 'force-dynamic';

const inter = Inter({ subsets: ["latin"] });

export const metadata: Metadata = {
  title: "AmmanGate - AI Home Cyber Bodyguard",
  description: "Local network security guardian for your home and small business",
  icons: {
    icon: "/logo.png",
    apple: "/logo.png",
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="dark">
      <body className={inter.className}>{children}</body>
    </html>
  );
}
