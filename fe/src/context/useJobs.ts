import { useContext } from "react";
import { JobContext } from "./job.context";

export function useJobs() {
  const context = useContext(JobContext);
  if (context === undefined) {
    throw new Error("useJobs must be used within a JobProvider");
  }

  return context;
}
