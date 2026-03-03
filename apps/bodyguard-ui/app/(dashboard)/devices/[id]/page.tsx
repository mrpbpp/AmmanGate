"use client";

import { useEffect, useState } from "react";
import { useRouter, useParams } from "next/navigation";
import { getCoreClient, type DeviceDetail } from "@/lib/core";

interface DeviceFingerprint {
	mac_vendor: string;
	os_guess: string;
	os_confidence: number;
	open_ports: number[];
	services: Record<number, string>;
	http_fingerprint?: {
		server: string;
		technologies: string[];
		title: string;
		status_code: number;
		has_auth: boolean;
		is_https: boolean;
		security_headers: string[];
	};
	last_scan: string;
	scan_status: string;
}

interface DeviceActivity {
	total_events: number;
	alerts_count: number;
	last_activity: string;
	first_seen: string;
	connection_count: number;
}

export default function DeviceDetailPage() {
	const router = useRouter();
	const params = useParams();
	const deviceId = params.id as string;

	const [device, setDevice] = useState<DeviceDetail | null>(null);
	const [fingerprint, setFingerprint] = useState<DeviceFingerprint | null>(null);
	const [loading, setLoading] = useState(true);
	const [scanning, setScanning] = useState(false);

	const client = getCoreClient();

	useEffect(() => {
		loadDevice();
	}, [deviceId]);

	const loadDevice = async () => {
		setLoading(true);
		try {
			const data = await client.getDevice(deviceId);
			setDevice(data);
		} catch (error) {
			console.error("Failed to load device:", error);
		} finally {
			setLoading(false);
		}
	};

	const runFingerprint = async () => {
		setScanning(true);
		try {
			const response = await fetch(`/api/core/v1/devices/${deviceId}/fingerprint`, {
				method: "POST",
			});
			if (response.ok) {
				const data = await response.json();
				setFingerprint(data.fingerprint);
			}
		} catch (error) {
			console.error("Fingerprint failed:", error);
		} finally {
			setScanning(false);
		}
	};

	const getRiskColor = (score: number) => {
		if (score >= 70) return "text-red-400";
		if (score >= 40) return "text-yellow-400";
		if (score >= 20) return "text-orange-400";
		return "text-emerald-400";
	};

	const getRiskLabel = (score: number) => {
		if (score >= 70) return "Critical";
		if (score >= 40) return "High";
		if (score >= 20) return "Medium";
		return "Low";
	};

	const getDeviceIcon = (type: string) => {
		const icons: Record<string, string> = {
			mobile: "📱",
			laptop: "💻",
			desktop: "🖥️",
			tablet: "📟",
			iot: "🔌",
			router: "📡",
			unknown: "📦",
		};
		return icons[type] || "📦";
	};

	const getOSIcon = (os: string) => {
		const icons: Record<string, string> = {
			Windows: "🪟",
			Linux: "🐧",
			macOS: "🍎",
			Android: "🤖",
			iOS: "📱",
			Router: "📡",
			IoT: "🔌",
			Unknown: "❓",
		};
		return icons[os] || "❓";
	};

	if (loading) {
		return (
			<div className="flex items-center justify-center py-20">
				<div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
			</div>
		);
	}

	if (!device) {
		return (
			<div className="card text-center py-20">
				<div className="text-6xl mb-4">🔍</div>
				<h2 className="text-xl font-semibold text-white mb-2">Device Not Found</h2>
				<p className="text-slate-400">The requested device does not exist.</p>
				<button onClick={() => router.back()} className="btn btn-primary mt-4">
					Go Back
				</button>
			</div>
		);
	}

	return (
		<div className="space-y-6">
			{/* Header */}
			<div className="flex items-center justify-between">
				<div>
					<button
						onClick={() => router.back()}
						className="text-slate-400 hover:text-white mb-2 flex items-center gap-2"
					>
						← Back to Devices
					</button>
					<h1 className="text-3xl font-bold text-white flex items-center gap-3">
						<span>{getDeviceIcon(device.type_guess)}</span>
						{device.hostname || "Unknown Device"}
					</h1>
					<p className="text-slate-400 mt-1">{device.vendor || "Unknown vendor"}</p>
				</div>
			</div>

			{/* Device Overview */}
			<div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
				{/* Basic Info Card */}
				<div className="card lg:col-span-2">
					<h2 className="text-lg font-semibold text-white mb-4">Device Information</h2>
					<div className="grid grid-cols-2 md:grid-cols-3 gap-4">
						<div>
							<p className="text-sm text-slate-400">IP Address</p>
							<p className="text-white font-mono">{device.ip || "N/A"}</p>
						</div>
						<div>
							<p className="text-sm text-slate-400">MAC Address</p>
							<p className="text-white font-mono text-sm">{device.mac}</p>
						</div>
						<div>
							<p className="text-sm text-slate-400">Device Type</p>
							<p className="text-white flex items-center gap-2">
								{getDeviceIcon(device.type_guess)}
								{device.type_guess || "Unknown"}
							</p>
						</div>
						<div>
							<p className="text-sm text-slate-400">First Seen</p>
							<p className="text-white">{device.first_seen || "Unknown"}</p>
						</div>
						<div>
							<p className="text-sm text-slate-400">Last Seen</p>
							<p className="text-white">{device.last_seen || "Unknown"}</p>
						</div>
						<div>
							<p className="text-sm text-slate-400">Risk Score</p>
							<div className="flex items-center gap-2">
								<span className={`text-lg font-bold ${getRiskColor(device.risk_score)}`}>
									{device.risk_score}/100
								</span>
								<span className={`text-sm px-2 py-1 rounded ${getRiskColor(device.risk_score)} bg-slate-700`}>
									{getRiskLabel(device.risk_score)}
								</span>
							</div>
						</div>
					</div>

					{/* Tags */}
					{device.tags && device.tags.length > 0 && (
						<div className="mt-4">
							<p className="text-sm text-slate-400 mb-2">Tags</p>
							<div className="flex flex-wrap gap-2">
								{device.tags.map((tag, index) => (
									<span
										key={index}
										className="px-3 py-1 bg-slate-700 text-slate-300 rounded-full text-sm"
									>
										{tag}
									</span>
								))}
							</div>
						</div>
					)}
				</div>

				{/* Activity Stats Card */}
				<div className="card">
					<h2 className="text-lg font-semibold text-white mb-4">Activity</h2>
					{device.activity_stats ? (
						<div className="space-y-3">
							<div className="flex items-center justify-between">
								<span className="text-slate-400">Events</span>
								<span className="text-white font-semibold">{device.activity_stats.total_events}</span>
							</div>
							<div className="flex items-center justify-between">
								<span className="text-slate-400">Alerts</span>
								<span className="text-white font-semibold">{device.activity_stats.alerts_count}</span>
							</div>
							<div className="flex items-center justify-between">
								<span className="text-slate-400">Last Activity</span>
								<span className="text-white">{device.activity_stats.last_activity || "Never"}</span>
							</div>
						</div>
					) : (
						<p className="text-slate-500 text-sm">No activity data available</p>
					)}
				</div>
			</div>

			{/* Fingerprint Section */}
			<div className="card">
				<div className="flex items-center justify-between mb-4">
					<h2 className="text-lg font-semibold text-white">Device Fingerprint</h2>
					<button
						onClick={runFingerprint}
						disabled={scanning}
						className="btn btn-primary text-sm py-2 px-4 disabled:opacity-50"
					>
						{scanning ? (
							<>
								<span className="animate-spin inline-block mr-2">⟳</span>
								Scanning...
							</>
						) : (
							<>
								<span>🔍</span>
								{fingerprint ? "Scan Again" : "Run Scan"}
							</>
						)}
					</button>
				</div>

				{!fingerprint ? (
					<div className="text-center py-8">
						<p className="text-slate-500">No fingerprint data available.</p>
						<p className="text-slate-600 text-sm mt-2">
							Click "Run Scan" to discover open ports, services, and OS details.
						</p>
					</div>
				) : (
					<div className="space-y-6">
						{/* OS Detection */}
						<div className="bg-slate-800/50 rounded-lg p-4 border border-slate-700">
							<div className="flex items-center justify-between mb-2">
								<h3 className="text-white font-medium">Operating System</h3>
								<span className="text-xs text-slate-500">{fingerprint.last_scan}</span>
							</div>
							<div className="flex items-center gap-3">
								<span className="text-2xl">{getOSIcon(fingerprint.os_guess)}</span>
								<div>
									<p className="text-white font-medium">{fingerprint.os_guess}</p>
									<p className="text-sm text-slate-400">
										Confidence: {Math.round(fingerprint.os_confidence)}%
									</p>
								</div>
							</div>
						</div>

						{/* Open Ports & Services */}
						<div className="bg-slate-800/50 rounded-lg p-4 border border-slate-700">
							<h3 className="text-white font-medium mb-3">Open Ports & Services</h3>
							{!fingerprint.open_ports || fingerprint.open_ports.length === 0 ? (
								<p className="text-slate-500 text-sm">No open ports detected</p>
							) : (
								<div className="grid grid-cols-2 md:grid-cols-3 gap-2">
									{fingerprint.open_ports.map((port) => (
										<div
											key={port}
											className="bg-slate-700/50 rounded-lg p-3 border border-slate-600"
										>
											<div className="flex items-center justify-between mb-1">
												<span className="text-lg font-bold text-blue-400">{port}</span>
												{(port === 80 || port === 443 || port === 8080) && (
													<span className="text-xs px-2 py-0.5 bg-green-900/50 text-green-400 rounded">
														HTTP
													</span>
												)}
											</div>
											<p className="text-sm text-slate-300">{fingerprint.services[port] || "Unknown"}</p>
										</div>
									))}
								</div>
							)}
						</div>

						{/* HTTP Fingerprint */}
						{fingerprint.http_fingerprint && (
							<div className="bg-slate-800/50 rounded-lg p-4 border border-slate-700">
								<h3 className="text-white font-medium mb-3">HTTP Service</h3>
								<div className="space-y-2 text-sm">
									{fingerprint.http_fingerprint.server && (
										<div className="flex justify-between">
											<span className="text-slate-400">Server:</span>
											<span className="text-white">{fingerprint.http_fingerprint.server}</span>
										</div>
									)}
									{fingerprint.http_fingerprint.title && (
										<div className="flex justify-between">
											<span className="text-slate-400">Title:</span>
											<span className="text-white truncate">{fingerprint.http_fingerprint.title}</span>
										</div>
									)}
									{fingerprint.http_fingerprint.status_code && (
										<div className="flex justify-between">
											<span className="text-slate-400">Status:</span>
											<span className={`${
												fingerprint.http_fingerprint.status_code === 200 ? "text-green-400" :
												fingerprint.http_fingerprint.status_code === 401 || fingerprint.http_fingerprint.status_code === 403 ?
												"text-yellow-400" : "text-red-400"
											}`}>
												{fingerprint.http_fingerprint.status_code}
											</span>
										</div>
									)}
									{fingerprint.http_fingerprint.has_auth && (
										<div className="flex justify-between">
											<span className="text-slate-400">Authentication:</span>
											<span className="text-yellow-400">Required</span>
										</div>
									)}
									{fingerprint.http_fingerprint.technologies && fingerprint.http_fingerprint.technologies.length > 0 && (
										<div>
											<span className="text-slate-400 block mb-1">Technologies:</span>
											<div className="flex flex-wrap gap-1">
												{fingerprint.http_fingerprint.technologies.map((tech, index) => (
													<span
														key={index}
														className="px-2 py-1 bg-purple-900/50 text-purple-300 rounded text-xs"
													>
														{tech}
													</span>
												))}
											</div>
										</div>
									)}
									{fingerprint.http_fingerprint.security_headers.length > 0 && (
										<div>
											<span className="text-slate-400 block mb-1">Security Headers:</span>
											<div className="flex flex-wrap gap-1">
												{fingerprint.http_fingerprint.security_headers.map((header, index) => (
													<span
														key={index}
														className="px-2 py-1 bg-blue-900/50 text-blue-300 rounded text-xs"
													>
														{header.replace("X-", "")}
													</span>
												))}
											</div>
										</div>
									)}
								</div>
							</div>
						)}

						{/* MAC Vendor */}
						<div className="bg-slate-800/50 rounded-lg p-4 border border-slate-700">
							<h3 className="text-white font-medium mb-2">Hardware Vendor</h3>
							<p className="text-white">{fingerprint.mac_vendor}</p>
						</div>
					</div>
				)}
			</div>
		</div>
	);
}
