import {Routes, Route, Navigate} from 'react-router-dom'
import './App.css'
import {Layout} from './components/Layout'
import {ConsignmentScreen} from "./screens/ConsignmentScreen.tsx"
import {ConsignmentDetailScreen} from "./screens/ConsignmentDetailScreen.tsx"
import {TaskDetailScreen} from "./screens/TaskDetailScreen.tsx";
import {PreconsignmentScreen} from "./screens/PreconsignmentScreen.tsx"
import { SignedIn, SignedOut, SignInButton } from '@asgardeo/react'

function App() {
  return (
    <>
      <SignedOut>
        <div className="min-h-screen flex items-center justify-center bg-gray-50 p-6">
          <div className="w-full max-w-md rounded-xl border border-gray-200 bg-white p-8 shadow-sm">
            <h1 className="text-2xl font-semibold text-gray-900">Trader Portal</h1>
            <p className="mt-2 text-sm text-gray-600">Sign in to continue to your consignments.</p>
            <div className="mt-6 flex flex-col gap-3">
              <SignInButton />
              {/* <SignUpButton /> */}
            </div>
          </div>
        </div>
      </SignedOut>

      <SignedIn>
        <Routes>
          <Route element={<Layout/>}>
            <Route path="/" element={<Navigate to="/consignments" replace/>}/>
            <Route path="/consignments" element={<ConsignmentScreen/>}/>
            <Route path="/consignments/:consignmentId" element={<ConsignmentDetailScreen/>}/>
            <Route path="/consignments/:consignmentId/tasks/:taskId" element={<TaskDetailScreen/>}/>
            <Route path="/pre-consignments" element={<PreconsignmentScreen/>}/>
            <Route path="/pre-consignments/:preConsignmentId/tasks/:taskId" element={<TaskDetailScreen/>}/>
          </Route>
          <Route path="*" element={<Navigate to="/consignments" replace/>} />
        </Routes>
      </SignedIn>
    </>
  )
}

export default App