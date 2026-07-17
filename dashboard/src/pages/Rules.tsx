import { useEffect, useState, type FormEvent } from "react";
import { api, ApiError, type Rule } from "../lib/api";

interface RulesProps {
  apiKey: string;
}

const ALGORITHMS = ["fixed_window", "sliding_window", "token_bucket"];

const inputClass =
  "rounded-lg border border-neutral-300 px-2.5 py-1.5 text-sm outline-none transition-colors focus:border-teal-500 focus:ring-2 focus:ring-teal-500/20 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-100";

export function Rules({ apiKey }: RulesProps) {
  const [rules, setRules] = useState<Rule[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [name, setName] = useState("");
  const [algorithm, setAlgorithm] = useState(ALGORITHMS[0]);
  const [limit, setLimit] = useState("100");
  const [window, setWindow] = useState("60");
  const [submitting, setSubmitting] = useState(false);

  async function refresh() {
    try {
      setRules(await api.listRules(apiKey));
      setError(null);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to load rules");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function handleCreate(e: FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      await api.createRule(apiKey, {
        name: name.trim(),
        algorithm,
        limit: Number(limit),
        window: algorithm === "token_bucket" ? 0 : Number(window),
      });
      setName("");
      await refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to create rule");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete(ruleName: string) {
    try {
      await api.deleteRule(apiKey, ruleName);
      await refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to delete rule");
    }
  }

  return (
    <div className="space-y-5">
      <form
        onSubmit={handleCreate}
        className="flex flex-wrap items-end gap-3 rounded-xl border border-neutral-200 bg-white p-4 shadow-sm shadow-neutral-900/[0.02] dark:border-neutral-800 dark:bg-neutral-900"
      >
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-neutral-500 dark:text-neutral-400">
            Name
          </label>
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="api_default"
            required
            className={inputClass}
          />
        </div>
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-neutral-500 dark:text-neutral-400">
            Algorithm
          </label>
          <select
            value={algorithm}
            onChange={(e) => setAlgorithm(e.target.value)}
            className={inputClass}
          >
            {ALGORITHMS.map((a) => (
              <option key={a} value={a}>
                {a}
              </option>
            ))}
          </select>
        </div>
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-neutral-500 dark:text-neutral-400">
            Limit
          </label>
          <input
            type="number"
            min={1}
            value={limit}
            onChange={(e) => setLimit(e.target.value)}
            className={`w-24 ${inputClass}`}
          />
        </div>
        {algorithm !== "token_bucket" && (
          <div className="flex flex-col gap-1">
            <label className="text-xs font-medium text-neutral-500 dark:text-neutral-400">
              Window (s)
            </label>
            <input
              type="number"
              min={1}
              value={window}
              onChange={(e) => setWindow(e.target.value)}
              className={`w-24 ${inputClass}`}
            />
          </div>
        )}
        <button
          type="submit"
          disabled={submitting || !name.trim()}
          className="rounded-lg bg-teal-500 px-3.5 py-1.5 text-sm font-semibold text-neutral-900 transition-colors hover:bg-teal-400 disabled:opacity-50 disabled:hover:bg-teal-500"
        >
          Add rule
        </button>
      </form>

      {error && (
        <p className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700 dark:bg-red-950 dark:text-red-400">
          {error}
        </p>
      )}

      <div className="overflow-hidden rounded-xl border border-neutral-200 shadow-sm shadow-neutral-900/[0.02] dark:border-neutral-800">
        <table className="w-full text-sm">
          <thead className="border-b border-neutral-200 bg-neutral-50 text-left text-xs text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900 dark:text-neutral-400">
            <tr>
              <th className="px-4 py-2.5 font-medium">Name</th>
              <th className="px-4 py-2.5 font-medium">Algorithm</th>
              <th className="px-4 py-2.5 font-medium">Limit</th>
              <th className="px-4 py-2.5 font-medium">Window</th>
              <th className="px-4 py-2.5" />
            </tr>
          </thead>
          <tbody className="divide-y divide-neutral-200 dark:divide-neutral-800">
            {!loading && rules.length === 0 && (
              <tr>
                <td
                  colSpan={5}
                  className="px-4 py-8 text-center text-neutral-400"
                >
                  No rules yet — add one above
                </td>
              </tr>
            )}
            {rules.map((rule) => (
              <tr
                key={rule.name}
                className="bg-white transition-colors hover:bg-neutral-50 dark:bg-neutral-950 dark:hover:bg-neutral-900"
              >
                <td className="px-4 py-2.5 font-medium text-neutral-900 dark:text-neutral-100">
                  {rule.name}
                </td>
                <td className="px-4 py-2.5">
                  <span className="rounded-full bg-teal-500/10 px-2 py-0.5 text-xs font-medium text-teal-700 dark:text-teal-400">
                    {rule.algorithm}
                  </span>
                </td>
                <td className="px-4 py-2.5 tabular-nums text-neutral-600 dark:text-neutral-400">
                  {rule.limit}
                </td>
                <td className="px-4 py-2.5 tabular-nums text-neutral-600 dark:text-neutral-400">
                  {rule.algorithm === "token_bucket" ? "—" : `${rule.window}s`}
                </td>
                <td className="px-4 py-2.5 text-right">
                  <button
                    onClick={() => handleDelete(rule.name)}
                    className="text-xs font-medium text-red-600 hover:underline dark:text-red-400"
                  >
                    Delete
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
