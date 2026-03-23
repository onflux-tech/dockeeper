import { Routes, Route } from "react-router";
import Home from "@/pages/home";

function App() {
	return (
		<div className="dark min-h-screen bg-background text-foreground">
			<Routes>
				<Route path="/" element={<Home />} />
			</Routes>
		</div>
	);
}

export default App;
