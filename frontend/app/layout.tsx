import './globals.css';
import { Cormorant_Garamond, DM_Sans } from 'next/font/google';

const display = Cormorant_Garamond({
  subsets: ['latin'],
  variable: '--font-display',
  weight: ['400', '500', '600']
});

const body = DM_Sans({
  subsets: ['latin'],
  variable: '--font-body',
  weight: ['400', '500', '600']
});

export const metadata = {
  title: 'Arachne',
  description: 'Result-first web crawler with readable page content and crawl tree diagnostics.'
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${display.variable} ${body.variable}`}>
      <body>{children}</body>
    </html>
  );
}
