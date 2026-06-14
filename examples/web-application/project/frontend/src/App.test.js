import React from 'react';
import { render, screen } from '@testing-library/react';
import App from './App';

test('renders app title', () => {
  render(<App />);
  const titleElement = screen.getByText(/ADP Web Application Example/i);
  expect(titleElement).toBeInTheDocument();
});

test('renders server status section', () => {
  render(<App />);
  const statusElement = screen.getByText(/Server Status/i);
  expect(statusElement).toBeInTheDocument();
});

test('renders login form when not logged in', () => {
  render(<App />);
  const usernameInput = screen.getByPlaceholderText(/Username/i);
  const passwordInput = screen.getByPlaceholderText(/Password/i);
  expect(usernameInput).toBeInTheDocument();
  expect(passwordInput).toBeInTheDocument();
});
