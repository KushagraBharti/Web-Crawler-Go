import './globals.css';
import { Fraunces, Newsreader, Barlow_Condensed } from 'next/font/google';

const fraunces = Fraunces({
  subsets: ['latin'],
  variable: '--font-display',
  weight: ['400', '700', '900'],
  style: ['normal'],
});

const newsreader = Newsreader({
  subsets: ['latin'],
  variable: '--font-reading',
  style: ['normal', 'italic'],
  weight: ['400', '500'],
});

const barlowCondensed = Barlow_Condensed({
  subsets: ['latin'],
  variable: '--font-label',
  weight: ['400', '500', '600', '700'],
});

export const metadata = {
  title: 'Arachne',
  description: 'Result-first web crawler with readable page content and crawl tree diagnostics.',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${fraunces.variable} ${newsreader.variable} ${barlowCondensed.variable}`}>
      <body>{children}</body>
    </html>
  );
}
