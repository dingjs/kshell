// 从本机 SSH 客户端（Xshell、PuTTY 等）扫描候选并批量导入当前工作区。
import * as DialogPrimitive from '@radix-ui/react-dialog';
import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { TFunction } from 'i18next';
import {
  importSSHConnections,
  scanSSHClientConfigs,
} from '../lib/api';
import type { SshClientCandidate } from '../lib/api';
import { backendError } from '../lib/errors';
import { Button } from './ui/button';
import { Dialog } from './ui/dialog';
import { Skeleton } from './ui/skeleton';

const SOURCE_KEYS = [
  'sshconfig',
  'env',
  'spring',
  'deploy',
  'docs',
  'manual',
  'xshell',
  'securecrt',
  'mobaxterm',
  'putty',
];

function sourceLabel(source: string, t: TFunction): string {
  return SOURCE_KEYS.includes(source) ? t(`ui.ssh.source_${source}`) : source;
}

function candidateTarget(c: SshClientCandidate): string {
  let target = c.User ? `${c.User}@${c.Host}` : c.Host;
  const port = c.Port > 0 ? c.Port : 22;
  if (port !== 22) target += `:${port}`;
  return target;
}

function candidateKey(c: SshClientCandidate, index: number): string {
  return `${index}|${c.Host}|${c.User}|${c.Port}|${c.SourceFile}`;
}

function initialSelection(items: SshClientCandidate[]): Record<string, boolean> {
  const sel: Record<string, boolean> = {};
  items.forEach((c, i) => {
    if (c.Confidence === 'high') sel[candidateKey(c, i)] = true;
  });
  return sel;
}

export default function SshImportDialog({
  open,
  onOpenChange,
  wsPath,
  onImported,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  wsPath: string;
  onImported: () => void;
}) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [items, setItems] = useState<SshClientCandidate[] | null>(null);
  const [selected, setSelected] = useState<Record<string, boolean>>({});
  const [error, setError] = useState('');
  const [importing, setImporting] = useState(false);

  useEffect(() => {
    if (!open) {
      setItems(null);
      setSelected({});
      setError('');
      return;
    }
    let cancelled = false;
    setLoading(true);
    setError('');
    setItems(null);
    scanSSHClientConfigs()
      .then((list) => {
        if (cancelled) return;
        setItems(list);
        setSelected(initialSelection(list));
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(backendError(e));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [open]);

  const selectedItems = useMemo(() => {
    if (!items) return [];
    return items.filter((c, i) => selected[candidateKey(c, i)]);
  }, [items, selected]);

  const handleImport = async () => {
    if (selectedItems.length === 0 || importing) return;
    setImporting(true);
    setError('');
    try {
      await importSSHConnections(wsPath, selectedItems);
      onImported();
      onOpenChange(false);
    } catch (e: unknown) {
      setError(backendError(e) || t('ui.ssh.import_failed'));
    } finally {
      setImporting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange} className="w-[min(92vw,28rem)]">
      <DialogPrimitive.Title className="mb-1 text-sm font-medium">
        {t('ui.ssh.import_title')}
      </DialogPrimitive.Title>
      <p className="mb-2 text-xs text-muted-foreground">{t('ui.ssh.import_hint')}</p>

      {loading && (
        <div className="flex flex-col gap-2" aria-busy="true">
          <p className="text-xs text-muted-foreground">{t('ui.ssh.import_scanning')}</p>
          <Skeleton className="h-10 rounded border border-border" />
          <Skeleton className="h-10 rounded border border-border" />
        </div>
      )}

      {!loading && error && !items && (
        <p className="text-xs text-destructive">{error}</p>
      )}

      {!loading && items && (
        <>
          {items.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t('ui.ssh.import_empty')}</p>
          ) : (
            <ul
              aria-label={t('ui.ssh.import_list_aria')}
              className="max-h-64 divide-y divide-border overflow-auto rounded border border-border"
            >
              {items.map((c, i) => {
                const key = candidateKey(c, i);
                const label = c.Name.trim() || candidateTarget(c);
                return (
                  <li key={key} className="px-2.5 py-1.5 text-xs">
                    <label className="flex cursor-pointer items-start gap-2">
                      <input
                        type="checkbox"
                        className="mt-0.5 shrink-0"
                        checked={!!selected[key]}
                        disabled={importing}
                        aria-label={label}
                        onChange={(e) =>
                          setSelected((prev) => ({ ...prev, [key]: e.target.checked }))
                        }
                      />
                      <span className="min-w-0 flex-1">
                        <span className="block font-medium">{label}</span>
                        <span className="mt-0.5 block truncate text-muted-foreground">
                          {candidateTarget(c)}
                        </span>
                        <span className="mt-1 flex flex-wrap items-center gap-1.5">
                          <span
                            className="rounded-sm border border-border px-1.5 py-px font-mono text-[10px]"
                            title={c.SourceFile || undefined}
                          >
                            {sourceLabel(c.Source, t)}
                          </span>
                          {c.NeedsPassword && (
                            <span className="text-[10px] text-amber-600 dark:text-amber-400">
                              {t('ui.ssh.import_needs_password')}
                            </span>
                          )}
                        </span>
                      </span>
                    </label>
                  </li>
                );
              })}
            </ul>
          )}
          {error && items.length > 0 && (
            <p className="mt-2 text-xs text-destructive">{error}</p>
          )}
        </>
      )}

      <div className="mt-3 flex justify-end gap-2">
        <Button
          size="sm"
          variant="secondary"
          disabled={importing}
          onClick={() => onOpenChange(false)}
        >
          {t('ui.ssh.cancel')}
        </Button>
        <Button
          size="sm"
          disabled={importing || loading || !items?.length || selectedItems.length === 0}
          onClick={() => void handleImport()}
        >
          {t('ui.ssh.import_confirm')}
        </Button>
      </div>
    </Dialog>
  );
}
