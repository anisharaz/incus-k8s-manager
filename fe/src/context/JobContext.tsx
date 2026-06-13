import { type ReactNode, useEffect, useMemo, useState } from "react";
import { JobContext, type DemoJobInput, type Job } from "./job.context";

const jobsApiBase = "/api/v1/jobs";

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

      const response = await fetch(jobsApiBase);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data: { jobs: Job[] } = await response.json();
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

  const submitDemoJob = async ({ name, durationSeconds }: DemoJobInput) => {
    const response = await fetch(`${jobsApiBase}/demo`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ name, durationSeconds }),
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    const data: { job: Job } = await response.json();
    await refreshJobs();
    return data.job;
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
      value={{ jobs, activeJobs, loading, error, submitDemoJob, refreshJobs }}
    >
      {children}
    </JobContext.Provider>
  );
}
