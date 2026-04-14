import { RunForm } from '@/components/RunForm';

export default function HomePage() {
  return (
    <main className="home-shell">
      <section className="home-hero">
        <div className="home-copy">
          <div className="eyebrow">Arachne reset</div>
          <h1>Find a page. Read what the crawler found. Verify it in JSON.</h1>
          <p>
            This version strips out the noisy telemetry-first dashboard and puts crawl results
            first. Start from a URL or keyword, inspect readable content, and watch the crawl tree
            grow from the root.
          </p>
          <div className="hero-points">
            <div><span>Seed</span><strong>URL or keyword</strong></div>
            <div><span>Output</span><strong>Readable pages</strong></div>
            <div><span>Verification</span><strong>Local JSON artifacts</strong></div>
          </div>
        </div>
        <RunForm />
      </section>
    </main>
  );
}
