import { useEffect, useState, type FormEvent } from "react";
import { api, ApiError, type Exemption } from "../lib/api";

interface ExemptionsProps {
  apiKey: string;
}

const inputClass =
  "rounded-lg border border-neutral-300 px-2.5 py-1.5 text-sm outline-none transition-colors focus:border-teal-500 focus:ring-2 focus:ring-teal-500/20 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-100";

export function Exemptions({ apiKey }: ExemptionsProps) {
  const [exemptions, setExemptions] = useState<Exemption[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [identifier, setIdentifier] = useState("");
  const [reason, setReason] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function refresh() {
    try {
      setExemptions(await api.listExemptions(apiKey));
      setError(null);
    } catch (err) {
      setError(
        err instanceof ApiError ? err.message : "Failed to load exemptions",
      );
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
      await api.createExemption(apiKey, {
        identifier: identifier.trim(),
        reason: reason.trim(),
      });
      setIdentifier("");
      setReason("");
      await refresh();
    } catch (err) {
      setError(
        err instanceof ApiError ? err.message : "Failed to create exemption",
      );
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete(id: string) {
    try {
      await api.deleteExemption(apiKey, id);
      await refresh();
    } catch (err) {
      setError(
        err instanceof ApiError ? err.message : "Failed to delete exemption",
      );
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
            Identifier
          </label>
          <input
            value={identifier}
            onChange={(e) => setIdentifier(e.target.value)}
            placeholder="internal-service"
            required
            className={inputClass}
          />
        </div>
        <div className="flex flex-1 flex-col gap-1">
          <label className="text-xs font-medium text-neutral-500 dark:text-neutral-400">
            Reason
          </label>
          <input
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder="Internal microservice, no rate limiting needed"
            className={`w-full ${inputClass}`}
          />
        </div>
        <button
          type="submit"
          disabled={submitting || !identifier.trim()}
          className="rounded-lg bg-teal-500 px-3.5 py-1.5 text-sm font-semibold text-neutral-900 transition-colors hover:bg-teal-400 disabled:opacity-50 disabled:hover:bg-teal-500"
        >
          Add exemption
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
              <th className="px-4 py-2.5 font-medium">Identifier</th>
              <th className="px-4 py-2.5 font-medium">Reason</th>
              <th className="px-4 py-2.5" />
            </tr>
          </thead>
          <tbody className="divide-y divide-neutral-200 dark:divide-neutral-800">
            {!loading && exemptions.length === 0 && (
              <tr>
                <td
                  colSpan={3}
                  className="px-4 py-8 text-center text-neutral-400"
                >
                  No exemptions yet — add one above
                </td>
              </tr>
            )}
            {exemptions.map((ex) => (
              <tr
                key={ex.identifier}
                className="bg-white transition-colors hover:bg-neutral-50 dark:bg-neutral-950 dark:hover:bg-neutral-900"
              >
                <td className="px-4 py-2.5 font-medium text-neutral-900 dark:text-neutral-100">
                  {ex.identifier}
                </td>
                <td className="px-4 py-2.5 text-neutral-600 dark:text-neutral-400">
                  {ex.reason}
                </td>
                <td className="px-4 py-2.5 text-right">
                  <button
                    onClick={() => handleDelete(ex.identifier)}
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
