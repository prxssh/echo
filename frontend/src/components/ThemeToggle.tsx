import React, { useEffect, useState } from 'react';

const STORAGE_KEY = 'theme';
type ThemeKey = 'dark' | 'light' | 'dark-slate' | 'light-paper' | 'dark-oled';

function applyTheme(theme: ThemeKey) {
  const root = document.documentElement;
  if (theme === 'dark') {
    root.removeAttribute('data-theme');
  } else {
    root.setAttribute('data-theme', theme);
  }
}

const OPTIONS: { key: ThemeKey; label: string }[] = [
  { key: 'dark', label: 'Dark (Default)' },
  { key: 'light', label: 'Light' },
  { key: 'dark-slate', label: 'Dark Slate' },
  { key: 'light-paper', label: 'Light Paper' },
  { key: 'dark-oled', label: 'Dark OLED' },
];

export default function ThemeToggle() {
  const [theme, setTheme] = useState<ThemeKey>('dark');

  useEffect(() => {
    const saved = (localStorage.getItem(STORAGE_KEY) as ThemeKey | null);
    if (saved) {
      setTheme(saved);
      applyTheme(saved);
      return;
    }
    const prefersLight = window.matchMedia?.('(prefers-color-scheme: light)').matches;
    const initial: ThemeKey = prefersLight ? 'light' : 'dark';
    setTheme(initial);
    applyTheme(initial);
  }, []);

  const onChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const value = e.target.value as ThemeKey;
    setTheme(value);
    applyTheme(value);
    localStorage.setItem(STORAGE_KEY, value);
  };

  return (
    <select className="input control" aria-label="Theme" value={theme} onChange={onChange}>
      {OPTIONS.map((o) => (
        <option key={o.key} value={o.key}>
          {o.label}
        </option>
      ))}
    </select>
  );
}
