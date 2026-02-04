import { DashboardClient } from '@/components/DashboardClient';

export default function RunPage({ params }: { params: { id: string } }) {
  return (
    <main>
      <DashboardClient runId={params.id} />
    </main>
  );
}