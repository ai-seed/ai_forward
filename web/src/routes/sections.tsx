import type { RouteObject } from 'react-router';

import { lazy, Suspense } from 'react';
import { Outlet } from 'react-router-dom';
import { varAlpha } from 'minimal-shared/utils';

import Box from '@mui/material/Box';
import LinearProgress, { linearProgressClasses } from '@mui/material/LinearProgress';

import { AuthLayout } from 'src/layouts/auth';
import { DashboardLayout } from 'src/layouts/dashboard';

import { ProtectedRoute } from 'src/components/auth/protected-route';

// ----------------------------------------------------------------------

export const RealDashboardPage = lazy(() => import('src/pages/real-dashboard'));
export const ApiKeysPage = lazy(() => import('src/pages/api-keys'));
export const ProfilePage = lazy(() => import('src/pages/profile'));
export const WalletPage = lazy(() => import('src/pages/wallet'));
export const ToolsPage = lazy(() => import('src/pages/tools'));
export const ModelsPage = lazy(() => import('src/pages/models'));
export const SignInPage = lazy(() => import('src/pages/sign-in'));
export const SignUpPage = lazy(() => import('src/pages/sign-up'));
export const ResetPasswordPage = lazy(() => import('src/pages/reset-password'));
export const OAuthCallbackPage = lazy(() => import('src/pages/auth/oauth-callback'));
export const Page404 = lazy(() => import('src/pages/page-not-found'));

const renderFallback = () => (
  <Box
    sx={{
      display: 'flex',
      flex: '1 1 auto',
      alignItems: 'center',
      justifyContent: 'center',
    }}
  >
    <LinearProgress
      sx={{
        width: 1,
        maxWidth: 320,
        bgcolor: (theme) => varAlpha(theme.vars.palette.text.primaryChannel, 0.16),
        [`& .${linearProgressClasses.bar}`]: { bgcolor: 'text.primary' },
      }}
    />
  </Box>
);

export const routesSection: RouteObject[] = [
  {
    element: (
      <ProtectedRoute>
        <DashboardLayout>
          <Suspense fallback={renderFallback()}>
            <Outlet />
          </Suspense>
        </DashboardLayout>
      </ProtectedRoute>
    ),
    children: [
      { index: true, element: <RealDashboardPage /> },
      { path: 'api-keys', element: <ApiKeysPage /> },
      { path: 'wallet', element: <WalletPage /> },
      { path: 'tools', element: <ToolsPage /> },
      { path: 'models', element: <ModelsPage /> },
      { path: 'profile', element: <ProfilePage /> },
    ],
  },
  {
    path: 'sign-in',
    element: (
      <AuthLayout>
        <SignInPage />
      </AuthLayout>
    ),
  },
  {
    path: 'sign-up',
    element: (
      <AuthLayout>
        <SignUpPage />
      </AuthLayout>
    ),
  },
  {
    path: 'reset-password',
    element: (
      <AuthLayout>
        <ResetPasswordPage />
      </AuthLayout>
    ),
  },
  {
    path: 'auth/oauth/callback',
    element: <OAuthCallbackPage />,
  },
  {
    path: '404',
    element: <Page404 />,
  },
  { path: '*', element: <Page404 /> },
];
