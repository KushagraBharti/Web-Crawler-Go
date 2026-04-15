import { RunForm } from '@/components/RunForm';

export default function HomePage() {
  return (
    <main className="home-shell">
      <header className="masthead anim-in">
        <span className="masthead-title">Arachne</span>
        <span className="masthead-meta">Result-first web crawler</span>
      </header>

      <div className="home-grid">
        <div className="home-copy">
          <span className="home-kicker anim-in anim-in-1">Web intelligence</span>
          <h1 className="home-headline anim-in anim-in-2">
            The web,<br />
            <em>read.</em>
          </h1>
          <p className="home-desc anim-in anim-in-3">
            Type a URL or a search phrase. Arachne resolves the seed, crawls
            outward, and extracts readable text from every page it discovers.
            Browse the content, trace the discovery tree, inspect the JSON.
          </p>
          <div className="home-features anim-in anim-in-4">
            <div className="home-feature">
              <span className="home-feature__label">Entry</span>
              <span className="home-feature__val">URL or keyword</span>
            </div>
            <div className="home-feature">
              <span className="home-feature__label">Output</span>
              <span className="home-feature__val">Readable pages</span>
            </div>
            <div className="home-feature">
              <span className="home-feature__label">Artifact</span>
              <span className="home-feature__val">Local JSON</span>
            </div>
          </div>
        </div>

        <div className="form-column anim-in anim-in-2">
          <RunForm />
        </div>
      </div>
    </main>
  );
}
