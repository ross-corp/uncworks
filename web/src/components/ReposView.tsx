export default function ReposView({
  repos,
}: {
  repos: { name: string; url: string }[];
}) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm" style={{ tableLayout: "fixed" }}>
        <colgroup>
          <col style={{ width: 200 }} />
          <col />
          <col style={{ width: 120 }} />
        </colgroup>
        <thead>
          <tr className="border-b border-edge text-left text-xs font-medium text-txt-tertiary">
            <th className="px-4 py-2">Name</th>
            <th className="px-4 py-2">URL</th>
            <th className="px-4 py-2">Actions</th>
          </tr>
        </thead>
        <tbody>
          {repos.map((repo) => (
            <tr
              key={repo.url}
              className="group border-b border-edge transition-colors hover:bg-surface-1"
            >
              <td className="px-4 py-2.5 font-medium text-txt-primary">
                {repo.name}
              </td>
              <td className="px-4 py-2.5 font-mono text-xs text-txt-secondary overflow-hidden text-ellipsis whitespace-nowrap">
                {repo.url}
              </td>
              <td className="px-4 py-2.5">
                <button className="btn-ghost px-3 py-1 text-xs text-red-400 opacity-0 group-hover:opacity-100 transition-opacity">
                  Remove
                </button>
              </td>
            </tr>
          ))}
          <tr>
            <td colSpan={3} className="px-4 py-2.5">
              <button className="text-xs text-txt-tertiary hover:text-txt-secondary transition-colors">
                + Add Repository
              </button>
            </td>
          </tr>
        </tbody>
      </table>

      {repos.length === 0 && (
        <div className="px-6 py-12 text-center text-sm text-txt-tertiary">
          No repositories configured.
        </div>
      )}
    </div>
  );
}
