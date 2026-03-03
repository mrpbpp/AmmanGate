"use client";

import { useState, useEffect } from "react";

interface AIAnalysis {
  narrative: string;
  suspected_cause: string;
  recommended_actions: string[];
  confidence: number;
}

interface AIAnalysisProps {
  deviceCount: number;
  alertCount: number;
}

export function AIAnalysis({ deviceCount, alertCount }: AIAnalysisProps) {
  const [analysis, setAnalysis] = useState<AIAnalysis | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadAnalysis();
  }, [deviceCount, alertCount]);

  const loadAnalysis = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch("http://127.0.0.1:8787/v1/ai/analyze", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          question: "Analyze the overall security posture of this home network",
        }),
      });
      if (!response.ok) {
        throw new Error("Failed to get AI analysis");
      }
      const data = await response.json();
      setAnalysis(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load AI analysis");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="bg-white rounded-lg shadow p-6">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-gray-900 flex items-center gap-2">
          🤖 AI Security Analysis
        </h2>
        <button
          onClick={loadAnalysis}
          disabled={loading}
          className="text-purple-600 hover:text-purple-700 text-sm font-medium disabled:opacity-50"
        >
          {loading ? "Analyzing..." : "Refresh"}
        </button>
      </div>

      {loading && (
        <div className="flex items-center gap-3 py-4">
          <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-purple-600"></div>
          <span className="text-gray-600">Analyzing network security...</span>
        </div>
      )}

      {error && (
        <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
          <p className="text-yellow-800 text-sm">{error}</p>
          <p className="text-yellow-600 text-xs mt-2">
            Make sure LM Studio is running on http://localhost:1234
          </p>
        </div>
      )}

      {!loading && !error && analysis && (
        <div className="space-y-4">
          <div>
            <label className="text-sm font-semibold text-gray-700">Analysis</label>
            <div
              className="mt-1 text-gray-700 prose prose-sm max-w-none"
              dangerouslySetInnerHTML={{ __html: analysis.narrative.replace(/\n/g, "<br/>") }}
            />
          </div>

          {analysis.suspected_cause && (
            <div>
              <label className="text-sm font-semibold text-gray-700">Suspected Cause</label>
              <p className="mt-1 text-gray-700">{analysis.suspected_cause}</p>
            </div>
          )}

          {analysis.recommended_actions && analysis.recommended_actions.length > 0 && (
            <div>
              <label className="text-sm font-semibold text-gray-700">Recommended Actions</label>
              <ul className="mt-1 space-y-1">
                {analysis.recommended_actions.map((action, idx) => (
                  <li key={idx} className="text-gray-700 flex items-start gap-2">
                    <span className="text-purple-600">•</span>
                    <span>{action}</span>
                  </li>
                ))}
              </ul>
            </div>
          )}

          <div className="flex items-center gap-2 text-sm text-gray-600">
            <span>AI Confidence:</span>
            <div className="flex-1 bg-gray-200 rounded-full h-2 max-w-xs">
              <div
                className="bg-purple-600 h-2 rounded-full transition-all"
                style={{ width: `${analysis.confidence * 100}%` }}
              ></div>
            </div>
            <span className="font-medium">{Math.round(analysis.confidence * 100)}%</span>
          </div>
        </div>
      )}

      {!loading && !error && !analysis && (
        <div className="text-center py-8">
          <p className="text-gray-500">Click "Refresh" to analyze your network security</p>
        </div>
      )}
    </div>
  );
}
