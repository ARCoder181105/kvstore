import { Header } from "@/components/layout/Header";
import { StatsPanel } from "@/components/dashboard/StatsPanel";
import { KeysTable } from "@/components/dashboard/KeysTable";
import { AddKeyForm } from "@/components/dashboard/AddKeyForm";
import { EventStream } from "@/components/dashboard/EventStream";

export default function DashboardPage() {
  return (
    <div className="flex flex-col min-h-screen bg-zinc-950 text-zinc-100">
      <Header />
      <main className="flex-1 container mx-auto p-4 md:p-8 space-y-6">
        <div className="mb-8">
          <h1 className="text-2xl font-semibold tracking-tight text-white mb-2">
            Dashboard
          </h1>
          <p className="text-zinc-500 text-sm">
            Manage your keys and monitor events in real-time.
          </p>
        </div>

        <StatsPanel />

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 lg:h-[calc(100vh-250px)] lg:min-h-[500px]">
          <div className="lg:col-span-2 flex flex-col space-y-4 lg:h-full lg:overflow-hidden">
            <div className="flex-shrink-0">
              <AddKeyForm />
            </div>
            <div className="flex-1 lg:overflow-y-auto min-h-0 lg:pr-2 custom-scrollbar">
              <KeysTable />
            </div>
          </div>
          
          <div className="lg:col-span-1 lg:h-full h-[400px]">
            <EventStream />
          </div>
        </div>
      </main>
      
      {/* Add a tiny bit of custom CSS for scrollbars to keep the sleek look */}
      <style dangerouslySetInnerHTML={{__html: `
        .custom-scrollbar::-webkit-scrollbar {
          width: 6px;
        }
        .custom-scrollbar::-webkit-scrollbar-track {
          background: transparent;
        }
        .custom-scrollbar::-webkit-scrollbar-thumb {
          background-color: #3f3f46;
          border-radius: 20px;
        }
      `}} />
    </div>
  );
}
