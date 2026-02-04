import './globals.css';
import { Bodoni_Moda, Familjen_Grotesk } from 'next/font/google';

const display = Bodoni_Moda({
  subsets: ['latin'],
  variable: '--font-display',
  weight: ['400', '600', '700']
});

const body = Familjen_Grotesk({
  subsets: ['latin'],
  variable: '--font-body',
  weight: ['400', '500', '600']
});

export const metadata = {
  title: 'Web Crawler Control Panel',
  description: 'High-performance crawler with live observability.'
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${display.variable} ${body.variable}`}>
      <body>{children}</body>
    </html>
  );
}