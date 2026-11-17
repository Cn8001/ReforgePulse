// Windows' own health checks, as the Checks tool and Reports name them.
export const checks = [
  { id: 'sfc', title: 'System files', what: 'Compares Windows files against their known good copies (sfc /verifyonly).', time: '5 to 15 min' },
  { id: 'dism', title: 'Windows image', what: 'Checks the store Windows repairs itself from (DISM /ScanHealth).', time: '5 to 20 min' },
  { id: 'chkdsk', title: 'System drive', what: 'Checks the file system while Windows runs (chkdsk /scan).', time: '1 to 10 min' },
  { id: 'memtest', title: 'Memory', what: 'Opens Windows Memory Diagnostic, which tests the RAM after a restart you choose.', time: 'one restart' },
] as const

export const checkTitle = (id: string) => checks.find((c) => c.id === id)?.title ?? id
