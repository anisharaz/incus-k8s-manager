import { createContext } from "react";
import type { Job } from "@/lib/types";

export type { Job, JobStatus } from "@/lib/types";

export interface JobContextType {
  jobs: Job[];
  activeJobs: Job[];
  loading: boolean;
  error: string | null;
  refreshJobs: () => Promise<void>;
}

export const JobContext = createContext<JobContextType | undefined>(undefined);
