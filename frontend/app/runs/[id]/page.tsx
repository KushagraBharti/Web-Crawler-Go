import { DashboardClient } from '@/components/DashboardClient';

export const dynamic = 'force-dynamic';

export default async function RunPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <main>
      <DashboardClient runId={id} />
    </main>
  );
}
