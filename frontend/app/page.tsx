import { RunForm } from '@/components/RunForm';

export default function HomePage() {
  return (
    <main className="stagger">
      <section className="hero">
        <div className="hero-content">
          <h1>Arachne</h1>
          <p>
            A web crawler built for transparency. Watch queues fill and drain,
            see hosts throttle in real time, and understand every failure.
          </p>
        </div>
        <div className="hero-stats">
          <div className="hero-stat">
            <div className="hero-stat__label">Speed</div>
            <div className="hero-stat__value">30+ pages/sec</div>
          </div>
          <div className="hero-stat">
            <div className="hero-stat__label">Updates</div>
            <div className="hero-stat__value">Real-time</div>
          </div>
          <div className="hero-stat">
            <div className="hero-stat__label">Memory</div>
            <div className="hero-stat__value">Bounded</div>
          </div>
        </div>
      </section>

      <section className="grid">
        <div className="panel span-5">
          <span className="badge badge--accent">About</span>
          <h2 style={{ marginTop: '1rem' }}>How it works</h2>
          <ul className="feature-list">
            <li>Strict backpressure keeps memory predictable</li>
            <li>Redirects re-enter the scheduler for politeness</li>
            <li>Per-host fairness prevents domain starvation</li>
            <li>Connection reuse tracked live</li>
            <li>Circuit breakers isolate failing hosts</li>
          </ul>
        </div>

        <div className="span-7">
          <RunForm />
        </div>
      </section>
    </main>
  );
}
