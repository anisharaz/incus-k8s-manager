import { createContext } from "react";

export type JobStatus = "queued" | "running" | "succeeded" | "failed";

export interface Job {
  id: string;
  type: string;
  name?: string;
  status: JobStatus;
  progress: number;
  stage?: string;
  message?: string;
  result?: Record<string, unknown>;
  error?: string;
  durationSeconds?: number;
  metadata?: Record<string, string>;
  createdAt: string;
  updatedAt: string;
  completedAt?: string | null;
}

export interface DemoJobInput {
  name: string;
  durationSeconds: number;
}

export interface JobContextType {
  jobs: Job[];
  activeJobs: Job[];
  loading: boolean;
  error: string | null;
  submitDemoJob: (input: DemoJobInput) => Promise<Job>;
  refreshJobs: () => Promise<void>;
}

export const JobContext = createContext<JobContextType | undefined>(undefined);
