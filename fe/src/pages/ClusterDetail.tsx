import { useEffect, useState } from "react";
import { Link, useParams } from "react-router";
import { ArrowLeft, HardDrive, ServerCog, AlertCircle } from "lucide-react";

interface Cluster {
  id: string;
  name: string;
  status: string;
  message?: string;
  jobId?: string;
  ip?: string;
  createdAt: string;
  updatedAt: string;
}

interface Job {
  id: string;
  type: string;
  name: string;
  status: string;
  progress: number;
  stage: string;
  message: string;
  error?: string;
  createdAt: string;
  updatedAt: string;
}

export function ClusterDetail() {
  const { clusterId } = useParams();
  const [cluster, setCluster] = useState<Cluster | null>(null);
  const [job, setJob] = useState<Job | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchCluster = async () => {
    if (!clusterId) return;

    try {
      setLoading(true);
      setError(null);
      const response = await fetch(`/api/v1/clusters/${clusterId}`);
      if (!response.ok) {
        throw new Error("Failed to fetch cluster");
      }
      const data = await response.json();
      setCluster(data.cluster);

      // Fetch job status if cluster has a job
      if (data.cluster.jobId) {
        const jobResponse = await fetch(`/api/v1/jobs/${data.cluster.jobId}`);
        if (jobResponse.ok) {
          const jobData = await jobResponse.json();
          setJob(jobData.job);
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch cluster");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchCluster();
    // Refresh cluster and job every 3 seconds if job is still running
    const interval = setInterval(fetchCluster, 3000);
    return () => clearInterval(interval);
  }, [clusterId]);

  if (loading) {
    return (
      <div className="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm">
        <div className="space-y-4 animate-pulse">
          <div className="h-8 bg-slate-200 rounded w-1/3"></div>
          <div className="h-4 bg-slate-200 rounded w-1/2"></div>
        </div>
      </div>
    );
  }

  if (error || !cluster) {
    return (
      <div className="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm">
        <p className="text-sm uppercase tracking-[0.2em] text-slate-500">
          Cluster
        </p>
        <h2 className="mt-2 text-2xl font-semibold text-slate-900">
          Cluster not found
        </h2>
        <p className="mt-2 text-slate-600">
          {error ||
            "The cluster id you opened does not match any known cluster."}
        </p>
        <Link
          to="/clusters"
          className="mt-6 inline-flex items-center gap-2 rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-slate-700"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to clusters
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <Link
        to="/clusters"
        className="inline-flex items-center gap-2 text-sm font-medium text-slate-600 transition-colors hover:text-slate-900"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to clusters
      </Link>

      <section className="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm">
        <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
          <div>
            <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-slate-100 px-3 py-1 text-xs font-medium uppercase tracking-[0.2em] text-slate-600">
              <ServerCog className="h-3.5 w-3.5" />
              Cluster detail
            </div>
            <h2 className="text-3xl font-semibold tracking-tight text-slate-900">
              {cluster.name}
            </h2>
            <p className="mt-2 text-sm text-slate-600">
              Cluster ID: {cluster.id}
            </p>
          </div>
          <div className="rounded-2xl bg-slate-50 px-4 py-3 text-sm text-slate-600">
            <p className="flex items-center gap-2 font-medium text-slate-900 capitalize">
              Status: {cluster.status}
            </p>
            {cluster.ip && <p className="mt-1">Master IP: {cluster.ip}</p>}
          </div>
        </div>

        <div className="mt-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <DetailCard label="Status" value={cluster.status} />
          <DetailCard
            label="Created"
            value={new Date(cluster.createdAt).toLocaleString()}
          />
          {cluster.ip && <DetailCard label="Master IP" value={cluster.ip} />}
          <DetailCard
            label="Last Updated"
            value={new Date(cluster.updatedAt).toLocaleString()}
          />
        </div>

        {cluster.message && (
          <div className="mt-6 rounded-2xl bg-blue-50 border border-blue-200 p-4">
            <p className="text-sm text-blue-700">{cluster.message}</p>
          </div>
        )}
      </section>

      {job && (
        <section className="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm">
          <h3 className="text-lg font-semibold text-slate-900 mb-4">
            Creation Progress
          </h3>

          <div className="space-y-4">
            <div>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm font-medium text-slate-600">
                  {job.stage}
                </span>
                <span className="text-sm font-medium text-slate-900">
                  {job.progress}%
                </span>
              </div>
              <div className="w-full bg-slate-200 rounded-full h-2">
                <div
                  className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                  style={{ width: `${job.progress}%` }}
                ></div>
              </div>
            </div>

            <div>
              <p className="text-sm text-slate-600 font-medium">Status</p>
              <p className="mt-1 text-sm text-slate-900 capitalize">
                {job.status}
              </p>
            </div>

            <div>
              <p className="text-sm text-slate-600 font-medium">Message</p>
              <p className="mt-1 text-sm text-slate-900">{job.message}</p>
            </div>

            {job.error && (
              <div className="rounded-2xl bg-rose-50 border border-rose-200 p-4 flex gap-2">
                <AlertCircle className="h-5 w-5 text-rose-600 flex-shrink-0 mt-0.5" />
                <p className="text-sm text-rose-700">{job.error}</p>
              </div>
            )}
          </div>
        </section>
      )}
    </div>
  );
}

function DetailCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-slate-200 bg-slate-50 p-4">
      <p className="text-xs uppercase tracking-wide text-slate-400">{label}</p>
      <p className="mt-2 text-base font-medium text-slate-900">{value}</p>
    </div>
  );
}
