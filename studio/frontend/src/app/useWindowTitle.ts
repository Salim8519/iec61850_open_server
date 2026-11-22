import { useEffect } from 'react';

/** The window's own title bar names the open project, e.g. "Al Kamil 400kV - ScadaBench"
 *  (change 0232). The app window is a browser --app window, which shows document.title. */
export function useWindowTitle(projectName: string | undefined) {
  useEffect(() => {
    const name = projectName?.trim();
    document.title = name ? `${name} - ScadaBench` : 'ScadaBench';
  }, [projectName]);
}
