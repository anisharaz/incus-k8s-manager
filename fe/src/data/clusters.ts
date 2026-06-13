export interface ClusterSummary {
  id: string;
  name: string;
  status: "healthy" | "degraded" | "offline";
  version: string;
  nodes: number;
  createdAt: string;
  region: string;
}

export const clusters: ClusterSummary[] = [
  {
    id: "cluster-alpha",
    name: "Alpha Cluster",
    status: "healthy",
    version: "v1.29.4",
    nodes: 5,
    createdAt: "2026-05-10T09:30:00Z",
    region: "us-east-1",
  },
  {
    id: "cluster-bravo",
    name: "Bravo Cluster",
    status: "degraded",
    version: "v1.28.9",
    nodes: 3,
    createdAt: "2026-04-18T14:15:00Z",
    region: "eu-west-3",
  },
  {
    id: "cluster-charlie",
    name: "Charlie Cluster",
    status: "offline",
    version: "v1.27.11",
    nodes: 4,
    createdAt: "2026-03-02T06:45:00Z",
    region: "ap-southeast-1",
  },
];

export function getClusterById(clusterId: string) {
  return clusters.find((cluster) => cluster.id === clusterId);
}
