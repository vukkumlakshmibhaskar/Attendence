import { LiveRecognitionPanel } from "../components/LiveRecognitionPanel.jsx";

export function LiveRecognitionPage({ onIdentifyFrame }) {
  return (
    <div className="page">
      <header className="page-heading">
        <h1>Live Recognition</h1>
        <p>Open camera to identify enrolled faces.</p>
      </header>
      <LiveRecognitionPanel onIdentifyFrame={onIdentifyFrame} />
    </div>
  );
}
