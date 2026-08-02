import { type ReactNode, useEffect, useMemo, useState } from "react";
import { JobContext } from "./job.context";
import type { Job } from "@/lib/types";
import { api } from "@/lib/api";

function isActiveJob(job: Job) {
  return job.status === "queued" || job.status === "running";
}

export function JobProvider({ children }: { children: ReactNode }) {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refreshJobs = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await api.get<{ jobs: Job[] }>("/api/v1/jobs");
      setJobs(data.jobs ?? []);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to fetch jobs";
      setError(message);
      console.error("Jobs fetch error:", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    let isMounted = true;

    const pollJobs = async () => {
      if (!isMounted) {
        return;
      }
      await refreshJobs();
    };

    pollJobs();

    const interval = setInterval(pollJobs, 3000);

    return () => {
      isMounted = false;
      clearInterval(interval);
    };
  }, []);

  const activeJobs = useMemo(() => jobs.filter(isActiveJob), [jobs]);

  return (
    <JobContext.Provider
      value={{ jobs, activeJobs, loading, error, refreshJobs }}
    >
      {children}
    </JobContext.Provider>
  );
}
