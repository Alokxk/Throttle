import { useEffect, useState } from "react";
import { api, type Client } from "./lib/api";
import { Login } from "./pages/Login";
import { Layout, type Tab } from "./components/Layout";
import { Overview } from "./pages/Overview";
import { Rules } from "./pages/Rules";
import { Exemptions } from "./pages/Exemptions";

const STORAGE_KEY = "throttle_dashboard_api_key";

function App() {
  const [apiKey, setApiKey] = useState<string | null>(() =>
    sessionStorage.getItem(STORAGE_KEY),
  );
  const [client, setClient] = useState<Client | null>(null);
  const [tab, setTab] = useState<Tab>("overview");

  useEffect(() => {
    if (!apiKey) return;
    api
      .me(apiKey)
      .then(setClient)
      .catch(() => {
        sessionStorage.removeItem(STORAGE_KEY);
        setApiKey(null);
      });
  }, [apiKey]);

  function handleLogin(key: string) {
    sessionStorage.setItem(STORAGE_KEY, key);
    setApiKey(key);
  }

  function handleLogout() {
    sessionStorage.removeItem(STORAGE_KEY);
    setApiKey(null);
    setClient(null);
  }

  if (!apiKey || !client) {
    return <Login onLogin={handleLogin} />;
  }

  return (
    <Layout
      client={client}
      tab={tab}
      onTabChange={setTab}
      onLogout={handleLogout}
    >
      {tab === "overview" && <Overview apiKey={apiKey} client={client} />}
      {tab === "rules" && <Rules apiKey={apiKey} />}
      {tab === "exemptions" && <Exemptions apiKey={apiKey} />}
    </Layout>
  );
}

export default App;
