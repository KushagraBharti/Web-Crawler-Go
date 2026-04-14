import { RunWorkspace } from '@/components/RunWorkspace';
import { API_BASE, getRun } from '@/lib/api';

export const dynamic = 'force-dynamic';

export default async function RunPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  try {
    const initial = await getRun(id);
    return <RunWorkspace initial={initial} runId={id} />;
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Unknown API error';
    return (
      <main className="workspace">
        <section className="workspace-hero">
          <div>
            <div className="eyebrow">Run workspace</div>
            <h1>Backend unavailable</h1>
            <p>The results page could not reach the crawler API.</p>
          </div>
        </section>

        <section className="panel">
          <div className="panel-header-block">
            <div className="eyebrow">Connection error</div>
            <h3>Server-side fetch failed</h3>
          </div>
          <p>
            Run page <code>{id}</code> tried to load from <code>{API_BASE}</code> and failed with:
          </p>
          <pre className="code-block">{message}</pre>
          <p>
            Start the backend first, then refresh this page. If the backend is running on a different
            port or host, set <code>NEXT_PUBLIC_API_BASE</code> in the frontend environment.
          </p>
        </section>
      </main>
    );
  }
}
