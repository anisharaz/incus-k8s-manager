import { useState, type FormEvent } from "react";
import { Loader2, PlayCircle, RefreshCcw, Timer } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import { useJobs } from "@/context";

const statusStyles: Record<string, string> = {
  queued: "bg-slate-100 text-slate-700 border-slate-200",
  running: "bg-blue-100 text-blue-700 border-blue-200",
  succeeded: "bg-emerald-100 text-emerald-700 border-emerald-200",
  failed: "bg-rose-100 text-rose-700 border-rose-200",
};

function formatJobLabel(name?: string, type?: string) {
  if (name && name.trim().length > 0) {
    return name;
  }

  return type ?? "job";
}

export function TaskDashboard() {
  const { jobs, activeJobs, loading, error, submitDemoJob, refreshJobs } =
    useJobs();
  const [name, setName] = useState("Demo rollout");
  const [durationSeconds, setDurationSeconds] = useState(12);
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setSubmitting(true);
    setSubmitError(null);

    try {
      await submitDemoJob({ name, durationSeconds });
    } catch (submitErr) {
      const message =
        submitErr instanceof Error ? submitErr.message : "Failed to submit job";
      setSubmitError(message);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <section className="rounded-2xl border border-slate-200 bg-white shadow-sm">
      <div className="flex flex-col gap-4 border-b border-slate-200 p-6 md:flex-row md:items-center md:justify-between">
        <div>
          <h3 className="text-xl font-semibold text-slate-900">
            Long-running tasks
          </h3>
          <p className="mt-1 text-sm text-slate-600">
            Submit a demo task, then watch the backend update progress while the
            frontend polls for status.
          </p>
        </div>
        <div className="flex items-center gap-2 text-sm text-slate-600">
          <Timer className="h-4 w-4" />
          <span>{activeJobs.length} active</span>
        </div>
      </div>

      <div className="grid gap-6 p-6 lg:grid-cols-[360px_minmax(0,1fr)]">
        <form className="space-y-4" onSubmit={handleSubmit}>
          <div>
            <label className="mb-2 block text-sm font-medium text-slate-700">
              Task name
            </label>
            <Input
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder="Database migration"
            />
          </div>

          <div>
            <label className="mb-2 block text-sm font-medium text-slate-700">
              Duration in seconds
            </label>
            <Input
              type="number"
              min={5}
              max={60}
              value={durationSeconds}
              onChange={(event) =>
                setDurationSeconds(Number(event.target.value) || 0)
              }
            />
          </div>

          {(submitError || error) && (
            <div className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700">
              {submitError ?? error}
            </div>
          )}

          <div className="flex gap-3">
            <Button type="submit" disabled={submitting || loading}>
              {submitting ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Starting...
                </>
              ) : (
                <>
                  <PlayCircle className="mr-2 h-4 w-4" />
                  Start task
                </>
              )}
            </Button>
            <Button type="button" variant="outline" onClick={refreshJobs}>
              <RefreshCcw className="mr-2 h-4 w-4" />
              Refresh
            </Button>
          </div>
        </form>

        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <h4 className="text-sm font-semibold uppercase tracking-wide text-slate-500">
              Job queue
            </h4>
            <span className="text-sm text-slate-500">{jobs.length} total</span>
          </div>

          <div className="space-y-3">
            {jobs.length === 0 ? (
              <div className="rounded-xl border border-dashed border-slate-200 bg-slate-50 px-4 py-8 text-sm text-slate-500">
                No tasks yet. Start one to see live polling and progress
                updates.
              </div>
            ) : (
              jobs.map((job) => (
                <article
                  key={job.id}
                  className="rounded-xl border border-slate-200 bg-slate-50 p-4"
                >
                  <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
                    <div>
                      <div className="flex flex-wrap items-center gap-2">
                        <h5 className="font-medium text-slate-900">
                          {formatJobLabel(job.name, job.type)}
                        </h5>
                        <span
                          className={`rounded-full border px-2 py-0.5 text-xs font-medium ${statusStyles[job.status] ?? "bg-slate-100 text-slate-700 border-slate-200"}`}
                        >
                          {job.status}
                        </span>
                      </div>
                      <p className="mt-1 text-sm text-slate-600">
                        {job.message ?? job.stage ?? "Waiting for updates"}
                      </p>
                    </div>
                    <div className="text-right text-sm text-slate-500">
                      <p>{job.progress}%</p>
                      <p>{new Date(job.updatedAt).toLocaleTimeString()}</p>
                    </div>
                  </div>

                  <div className="mt-3 h-2 overflow-hidden rounded-full bg-slate-200">
                    <div
                      className="h-full rounded-full bg-gradient-to-r from-blue-500 via-cyan-500 to-emerald-500 transition-all"
                      style={{
                        width: `${Math.max(0, Math.min(100, job.progress))}%`,
                      }}
                    />
                  </div>

                  <div className="mt-3 flex flex-wrap gap-3 text-xs text-slate-500">
                    <span>ID: {job.id}</span>
                    <span>
                      Started: {new Date(job.createdAt).toLocaleString()}
                    </span>
                    {job.completedAt ? (
                      <span>
                        Completed: {new Date(job.completedAt).toLocaleString()}
                      </span>
                    ) : null}
                  </div>

                  {job.result ? (
                    <>
                      <Separator className="my-3" />
                      <pre className="overflow-x-auto rounded-lg bg-slate-900 px-4 py-3 text-xs text-slate-100">
                        {JSON.stringify(job.result, null, 2)}
                      </pre>
                    </>
                  ) : null}

                  {job.error ? (
                    <p className="mt-3 rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-700">
                      {job.error}
                    </p>
                  ) : null}
                </article>
              ))
            )}
          </div>
        </div>
      </div>
    </section>
  );
}
