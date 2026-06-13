export function Settings() {
  return (
    <div className="space-y-4">
      <div>
        <p className="text-sm uppercase tracking-[0.2em] text-slate-500">
          Incus Setting
        </p>
        <h2 className="mt-2 text-3xl font-semibold tracking-tight text-slate-900">
          Incus settings
        </h2>
      </div>
      <p className="max-w-2xl text-sm text-slate-600">
        Configure cluster-related Incus options here. This route is reserved for
        environment, access, and platform settings.
      </p>
    </div>
  );
}
