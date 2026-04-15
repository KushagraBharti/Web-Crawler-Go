import { RunWorkspace } from '@/components/RunWorkspace';
import { API_BASE, getRun } from '@/lib/api';

export const dynamic = 'force-dynamic';

export default async function RunPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;

  let initial;
  let errorMessage: string | null = null;

  try {
    initial = await getRun(id);
  } catch (error) {
    errorMessage = error instanceof Error ? error.message : 'Unknown error';
  }

  if (errorMessage || !initial) {
    return (
      <main className="error-shell">
        <div className="eyebrow" style={{ marginBottom: 20 }}>Connection error</div>
        <h1>Backend unavailable</h1>
        <p style={{ marginBottom: 12 }}>
          Run <code>{id}</code> could not be loaded from <code>{API_BASE}</code>.
        </p>
        <pre className="code-block">{errorMessage}</pre>
        <p style={{ marginTop: 12 }}>
          Start the backend, then refresh. If it&apos;s on a different port,
          set <code>NEXT_PUBLIC_API_BASE</code> in the frontend environment.
        </p>
      </main>
    );
  }

  return <RunWorkspace initial={initial} runId={id} />;
}
