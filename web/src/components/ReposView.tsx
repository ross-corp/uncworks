function repoName(url: string): string {
  const parts = url.replace(/\.git$/, "").split("/");
  return parts[parts.length - 1] || url;
}

export default function ReposView({
  repos,
}: {
  repos: string[];
}) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm" style={{ tableLayout: "fixed" }}>
        <colgroup>
          <col style={{ width: 200 }} />
          <col />
        </colgroup>
        <thead>
          <tr className="border-b border-edge text-left text-xs font-medium text-txt-tertiary">
            <th className="px-4 py-2">Name</th>
            <th className="px-4 py-2">URL</th>
          </tr>
        </thead>
        <tbody>
          {repos.map((url) => (
            <tr
              key={url}
              className="group border-b border-edge transition-colors hover:bg-surface-1"
            >
              <td className="px-4 py-2.5 font-medium text-txt-primary">
                {repoName(url)}
              </td>
              <td className="px-4 py-2.5 font-mono text-xs text-txt-secondary overflow-hidden text-ellipsis whitespace-nowrap">
                {url}
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {repos.length === 0 && (
        <div className="px-6 py-12 text-center text-sm text-txt-tertiary">
          No repositories found in any agent runs.
        </div>
      )}
    </div>
  );
}
