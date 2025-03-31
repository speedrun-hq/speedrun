import React from 'react';
import { BrowserRouter as Router, Routes, Route, Link } from 'react-router-dom';
import { Layout } from './components/Layout';
import { CreateIntentForm } from './components/CreateIntentForm';
import IntentList from './components/IntentList';

const Home: React.FC = () => (
  <div className="card">
    <h1 className="text-2xl font-bold mb-4">Welcome to ZetaFast</h1>
    <p className="mb-4">Cross-chain USDC transfer service</p>
    <div className="space-y-4">
      <div className="flex space-x-4">
        <Link to="/intents" className="btn btn-primary">View Intents</Link>
        <Link to="/create" className="btn btn-secondary">Create Intent</Link>
      </div>
      <div className="mt-8">
        <h2 className="text-xl font-semibold mb-4">Features</h2>
        <ul className="list-disc list-inside space-y-2">
          <li>Cross-chain USDC transfers</li>
          <li>Real-time transaction tracking</li>
          <li>Secure and reliable</li>
          <li>Easy to use interface</li>
        </ul>
      </div>
    </div>
  </div>
);

const App: React.FC = () => {
  return (
    <Router>
      <Layout>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/intents" element={<IntentList />} />
          <Route path="/create" element={<CreateIntentForm />} />
        </Routes>
      </Layout>
    </Router>
  );
};

export default App;
