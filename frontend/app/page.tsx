import { RunForm } from '@/components/RunForm';

export default function HomePage() {
  return (
    <main>
      <section className="hero">
        <div>
          <div className="badge">System Lab</div>
          <h1>Instrument-Panel Crawler</h1>
          <p>
            A fast, bounded crawler with live visibility into queues, host throttling, and failure modes.
            Treat the web like a hostile system and learn from every edge case.
          </p>
        </div>
        <div className="hero-card">
          <div className="hero-metric">
            <span>Throughput</span>
            <strong>30+ pages/sec</strong>
          </div>
          <div className="hero-metric">
            <span>Connection reuse</span>
            <strong>Measured live</strong>
          </div>
          <div className="hero-metric">
            <span>Queues</span>
            <strong>Always bounded</strong>
          </div>
        </div>
      </section>

      <section className="grid">
        <div className="panel span-5">
          <div className="badge">What it solves</div>
          <h2 style={{ marginTop: 12 }}>Crawl fast, stay safe</h2>
          <ul className="list">
            <li>Strict backpressure keeps memory predictable.</li>
            <li>Redirects re-enter the scheduler for politeness.</li>
            <li>Per-host fairness prevents hot domains from starving others.</li>
            <li>Connection reuse and latency tracked in real time.</li>
          </ul>
        </div>
        <div className="span-7">
          <RunForm />
        </div>
      </section>
    </main>
  );
}
