import { useEffect, useState } from 'react';

const getHash = () => window.location.hash.replace(/^#/, '') || '/';

export default function useHashRoute() {
  const [route, setRoute] = useState(getHash);

  useEffect(() => {
    const onChange = () => setRoute(getHash());
    window.addEventListener('hashchange', onChange);
    return () => window.removeEventListener('hashchange', onChange);
  }, []);

  const navigate = (to) => {
    window.location.hash = to;
  };

  return { route, navigate };
}
