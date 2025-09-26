import React, { Suspense } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { Helmet } from 'react-helmet-async'
import { useAuth } from '@/contexts/AuthContext'
import Layout from '@/components/Layout'
import LoadingSpinner from '@/components/LoadingSpinner'
import ErrorBoundary from '@/components/ErrorBoundary'

// Lazy load pages for better performance
const Dashboard = React.lazy(() => import('@/pages/Dashboard'))
const Intents = React.lazy(() => import('@/pages/Intents'))
const IntentDetail = React.lazy(() => import('@/pages/IntentDetail'))
const Slices = React.lazy(() => import('@/pages/Slices'))
const SliceDetail = React.lazy(() => import('@/pages/SliceDetail'))
const Infrastructure = React.lazy(() => import('@/pages/Infrastructure'))
const Experiments = React.lazy(() => import('@/pages/Experiments'))
const ExperimentDetail = React.lazy(() => import('@/pages/ExperimentDetail'))
const Settings = React.lazy(() => import('@/pages/Settings'))
const Login = React.lazy(() => import('@/pages/Login'))
const NotFound = React.lazy(() => import('@/pages/NotFound'))

// Protected Route Component
const ProtectedRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated, isLoading } = useAuth()

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <LoadingSpinner size="lg" />
      </div>
    )
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
}

// Public Route Component (redirect if authenticated)
const PublicRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated, isLoading } = useAuth()

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <LoadingSpinner size="lg" />
      </div>
    )
  }

  if (isAuthenticated) {
    return <Navigate to="/dashboard" replace />
  }

  return <>{children}</>
}

function App() {
  return (
    <ErrorBoundary>
      <Helmet>
        <title>O-RAN Intent-MANO Dashboard</title>
        <meta
          name="description"
          content="Real-time monitoring and management dashboard for O-RAN Intent-Based MANO system"
        />
      </Helmet>

      <Routes>
        {/* Public Routes */}
        <Route
          path="/login"
          element={
            <PublicRoute>
              <Suspense fallback={<LoadingSpinner size="lg" />}>
                <Login />
              </Suspense>
            </PublicRoute>
          }
        />

        {/* Protected Routes */}
        <Route
          path="/"
          element={
            <ProtectedRoute>
              <Layout />
            </ProtectedRoute>
          }
        >
          <Route index element={<Navigate to="/dashboard" replace />} />

          <Route
            path="dashboard"
            element={
              <Suspense fallback={<LoadingSpinner />}>
                <Dashboard />
              </Suspense>
            }
          />

          <Route path="intents">
            <Route
              index
              element={
                <Suspense fallback={<LoadingSpinner />}>
                  <Intents />
                </Suspense>
              }
            />
            <Route
              path=":id"
              element={
                <Suspense fallback={<LoadingSpinner />}>
                  <IntentDetail />
                </Suspense>
              }
            />
          </Route>

          <Route path="slices">
            <Route
              index
              element={
                <Suspense fallback={<LoadingSpinner />}>
                  <Slices />
                </Suspense>
              }
            />
            <Route
              path=":id"
              element={
                <Suspense fallback={<LoadingSpinner />}>
                  <SliceDetail />
                </Suspense>
              }
            />
          </Route>

          <Route
            path="infrastructure"
            element={
              <Suspense fallback={<LoadingSpinner />}>
                <Infrastructure />
              </Suspense>
            }
          />

          <Route path="experiments">
            <Route
              index
              element={
                <Suspense fallback={<LoadingSpinner />}>
                  <Experiments />
                </Suspense>
              }
            />
            <Route
              path=":id"
              element={
                <Suspense fallback={<LoadingSpinner />}>
                  <ExperimentDetail />
                </Suspense>
              }
            />
          </Route>

          <Route
            path="settings"
            element={
              <Suspense fallback={<LoadingSpinner />}>
                <Settings />
              </Suspense>
            }
          />
        </Route>

        {/* 404 Route */}
        <Route
          path="*"
          element={
            <Suspense fallback={<LoadingSpinner size="lg" />}>
              <NotFound />
            </Suspense>
          }
        />
      </Routes>
    </ErrorBoundary>
  )
}

export default App