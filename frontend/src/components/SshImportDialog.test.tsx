// SshImportDialog：扫描候选、默认勾选高置信度、导入回调。
import '@testing-library/jest-dom/vitest';
import { act, cleanup, fireEvent, render, screen, within } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import SshImportDialog from './SshImportDialog';
import type { SshClientCandidate, SshConnection } from '../lib/api';
import { tt } from '../test/i18n';

const mocks = vi.hoisted(() => ({
  scanSSHClientConfigs: vi.fn(),
  importSSHConnections: vi.fn(),
}));
vi.mock('../lib/api', () => mocks);

const candidates: SshClientCandidate[] = [
  {
    Name: 'Prod',
    Host: '10.0.0.1',
    User: 'root',
    Port: 22,
    IdentityFile: '',
    Source: 'xshell',
    SourceFile: 'C:\\xsh\\prod.xsh',
    Password: '',
    NeedsPassword: false,
    Confidence: 'high',
  },
  {
    Name: '',
    Host: 'jump.example.com',
    User: 'ops',
    Port: 2222,
    IdentityFile: '',
    Source: 'putty',
    SourceFile: '',
    Password: '',
    NeedsPassword: true,
    Confidence: 'medium',
  },
];

afterEach(cleanup);

beforeEach(() => {
  vi.clearAllMocks();
  mocks.scanSSHClientConfigs.mockResolvedValue(candidates);
  mocks.importSSHConnections.mockResolvedValue([] as SshConnection[]);
});

describe('SshImportDialog', () => {
  it('打开时扫描并默认勾选 high 置信度', async () => {
    render(
      <SshImportDialog
        open
        onOpenChange={() => {}}
        wsPath="D:\\proj-a"
        onImported={() => {}}
      />,
    );

    expect(mocks.scanSSHClientConfigs).toHaveBeenCalled();
    expect(await screen.findByText('Prod')).toBeInTheDocument();

    const prod = screen.getByRole('checkbox', { name: 'Prod' }) as HTMLInputElement;
    const jump = screen.getByRole('checkbox', { name: 'ops@jump.example.com:2222' }) as HTMLInputElement;
    expect(prod.checked).toBe(true);
    expect(jump.checked).toBe(false);
  });

  it('展示来源与 NeedsPassword 提示', async () => {
    render(
      <SshImportDialog
        open
        onOpenChange={() => {}}
        wsPath="D:\\proj-a"
        onImported={() => {}}
      />,
    );

    await screen.findByText('Prod');
    expect(screen.getByText(tt('ui.ssh.source_xshell'))).toBeInTheDocument();
    expect(screen.getByText(tt('ui.ssh.source_putty'))).toBeInTheDocument();
    expect(screen.getByText(tt('ui.ssh.import_needs_password'))).toBeInTheDocument();
  });

  it('确认导入调用 API 并触发 onImported', async () => {
    const onImported = vi.fn();
    const onOpenChange = vi.fn();
    render(
      <SshImportDialog
        open
        onOpenChange={onOpenChange}
        wsPath="D:\\proj-a"
        onImported={onImported}
      />,
    );

    await screen.findByText('Prod');
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: tt('ui.ssh.import_confirm') }));
    });

    expect(mocks.importSSHConnections).toHaveBeenCalledWith(
      expect.stringMatching(/proj-a$/i),
      [candidates[0]],
    );
    expect(onImported).toHaveBeenCalled();
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it('扫描结果为空时显示空态', async () => {
    mocks.scanSSHClientConfigs.mockResolvedValue([]);
    render(
      <SshImportDialog
        open
        onOpenChange={() => {}}
        wsPath="D:\\proj-a"
        onImported={() => {}}
      />,
    );

    expect(await screen.findByText(tt('ui.ssh.import_empty'))).toBeInTheDocument();
    const confirm = screen.getByRole('button', { name: tt('ui.ssh.import_confirm') });
    expect(confirm).toBeDisabled();
  });

  it('可多选后一并导入', async () => {
    render(
      <SshImportDialog
        open
        onOpenChange={() => {}}
        wsPath="D:\\ws"
        onImported={() => {}}
      />,
    );

    await screen.findByText('Prod');
    const list = screen.getByLabelText(tt('ui.ssh.import_list_aria'));
    const jump = within(list).getByRole('checkbox', { name: 'ops@jump.example.com:2222' });
    fireEvent.click(jump);

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: tt('ui.ssh.import_confirm') }));
    });

    expect(mocks.importSSHConnections).toHaveBeenCalledWith(
      expect.stringMatching(/\\ws$/i),
      candidates,
    );
  });
});
