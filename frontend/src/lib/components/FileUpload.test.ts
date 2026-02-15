import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, fireEvent, screen, cleanup } from '@testing-library/svelte';
import FileUpload from './FileUpload.svelte';

describe('FileUpload', () => {
	beforeEach(() => {
		// Mock URL.createObjectURL
		global.URL.createObjectURL = vi.fn(() => 'blob:mock-url');
		global.URL.revokeObjectURL = vi.fn();
	});

	afterEach(() => {
		cleanup();
		vi.clearAllMocks();
	});

	function createMockFile(name: string, size: number, type: string): File {
		const blob = new Blob(['x'.repeat(size)], { type });
		return new File([blob], name, { type });
	}

	describe('rendering', () => {
		it('renders drop zone with empty state', () => {
			render(FileUpload);
			expect(screen.getByText(/Drag and drop files here/i)).toBeInTheDocument();
			expect(screen.getByText(/click to browse/i)).toBeInTheDocument();
		});

		it('shows compact text when compact prop is true', () => {
			render(FileUpload, { props: { compact: true } });
			expect(screen.getByText(/Drop files or/i)).toBeInTheDocument();
			expect(screen.getByText('browse')).toBeInTheDocument();
		});

		it('shows file limits in hint text', () => {
			render(FileUpload, { props: { maxFiles: 5, maxFileSize: 10 * 1024 * 1024 } });
			expect(screen.getByText(/Max 5 files/i)).toBeInTheDocument();
			expect(screen.getByText(/10 MB/i)).toBeInTheDocument();
		});

		it('applies disabled state correctly', () => {
			render(FileUpload, { props: { disabled: true } });
			const dropZone = screen.getByRole('button', { name: /Click or drag files to upload/i });
			expect(dropZone).toHaveAttribute('aria-disabled', 'true');
		});
	});

	describe('file selection', () => {
		// Skip - needs Svelte 5 event prop migration (createEventDispatcher events don't work with DOM listeners)
		it.skip('dispatches select event when files are added via input', async () => {
			const { container } = render(FileUpload);
			const selectHandler = vi.fn();
			
			// Get the component wrapper and add event listener
			const wrapper = container.querySelector('.file-upload');
			wrapper?.addEventListener('select', selectHandler as EventListener);

			const fileInput = container.querySelector('input[type="file"]') as HTMLInputElement;
			const mockFile = createMockFile('test.jpg', 1024, 'image/jpeg');
			
			Object.defineProperty(fileInput, 'files', {
				value: [mockFile],
				writable: false
			});

			await fireEvent.change(fileInput);

			expect(selectHandler).toHaveBeenCalledTimes(1);
		});

		it('shows file previews after selection', () => {
			render(FileUpload, {
				props: {
					files: [createMockFile('image.png', 1024, 'image/png')]
				}
			});

			expect(screen.getByText('image.png')).toBeInTheDocument();
			expect(screen.getByText('1 KB')).toBeInTheDocument();
		});

		it('prevents duplicate files', async () => {
			const file1 = createMockFile('test.jpg', 1024, 'image/jpeg');
			const { container } = render(FileUpload, {
				props: { files: [file1] }
			});

			const fileInput = container.querySelector('input[type="file"]') as HTMLInputElement;
			Object.defineProperty(fileInput, 'files', {
				value: [createMockFile('test.jpg', 1024, 'image/jpeg')],
				writable: false
			});

			await fireEvent.change(fileInput);

			// Should show error about duplicate
			expect(screen.getByRole('alert')).toHaveTextContent(/already added/i);
		});
	});

	describe('file validation', () => {
		it('shows error when file exceeds max size', async () => {
			const { container } = render(FileUpload, { props: { maxFileSize: 1024 } }); // 1KB max

			const fileInput = container.querySelector('input[type="file"]') as HTMLInputElement;
			const largeFile = createMockFile('large.jpg', 2048, 'image/jpeg'); // 2KB file

			Object.defineProperty(fileInput, 'files', {
				value: [largeFile],
				writable: false
			});

			await fireEvent.change(fileInput);

			expect(screen.getByRole('alert')).toHaveTextContent(/exceeds maximum size/i);
		});

		it('shows error when max files limit exceeded', async () => {
			const existingFiles = [
				createMockFile('1.jpg', 100, 'image/jpeg'),
				createMockFile('2.jpg', 100, 'image/jpeg')
			];
			
			const { container } = render(FileUpload, { props: { files: existingFiles, maxFiles: 2 } });

			const fileInput = container.querySelector('input[type="file"]') as HTMLInputElement;
			Object.defineProperty(fileInput, 'files', {
				value: [createMockFile('3.jpg', 100, 'image/jpeg')],
				writable: false
			});

			await fireEvent.change(fileInput);

			expect(screen.getByRole('alert')).toHaveTextContent(/Maximum 2 files/i);
		});

		// Skip - Svelte 5 transitions with fade cause timing issues in jsdom
		it.skip('clears error when dismiss button clicked', async () => {
			const { container } = render(FileUpload, { props: { maxFileSize: 100 } });

			const fileInput = container.querySelector('input[type="file"]') as HTMLInputElement;
			Object.defineProperty(fileInput, 'files', {
				value: [createMockFile('large.jpg', 1000, 'image/jpeg')],
				writable: false
			});

			await fireEvent.change(fileInput);
			expect(screen.getByRole('alert')).toBeInTheDocument();

			const dismissBtn = screen.getByLabelText(/dismiss error/i);
			await fireEvent.click(dismissBtn);

			expect(screen.queryByRole('alert')).not.toBeInTheDocument();
		});
	});

	describe('file removal', () => {
		// Skip - needs Svelte 5 event prop migration (createEventDispatcher events don't work with DOM listeners)
		it.skip('dispatches remove event when file removed', async () => {
			const file = createMockFile('test.jpg', 1024, 'image/jpeg');
			const { container } = render(FileUpload, { props: { files: [file] } });
			
			const removeHandler = vi.fn();
			const wrapper = container.querySelector('.file-upload');
			wrapper?.addEventListener('remove', removeHandler as EventListener);

			const removeBtn = screen.getByLabelText(/Remove test.jpg/i);
			await fireEvent.click(removeBtn);

			expect(removeHandler).toHaveBeenCalledTimes(1);
		});

		it('hides remove button during upload', () => {
			render(FileUpload, {
				props: {
					files: [createMockFile('test.jpg', 1024, 'image/jpeg')],
					uploading: true
				}
			});

			expect(screen.queryByLabelText(/Remove/i)).not.toBeInTheDocument();
		});
	});

	describe('drag and drop', () => {
		it('shows drag overlay on dragenter', async () => {
			render(FileUpload);
			const dropZone = screen.getByRole('button', { name: /Click or drag files to upload/i });

			const dataTransfer = { types: ['Files'], files: [] };
			await fireEvent.dragEnter(dropZone, { dataTransfer });

			expect(screen.getByText('Drop files here')).toBeInTheDocument();
		});

		// Skip - Svelte 5 transitions with fade cause timing issues in jsdom
		it.skip('hides drag overlay on dragleave', async () => {
			render(FileUpload);
			const dropZone = screen.getByRole('button', { name: /Click or drag files to upload/i });

			const dataTransfer = { types: ['Files'], files: [] };
			await fireEvent.dragEnter(dropZone, { dataTransfer });
			await fireEvent.dragLeave(dropZone);

			expect(screen.queryByText('Drop files here')).not.toBeInTheDocument();
		});
	});

	describe('upload progress', () => {
		it('shows progress bar during upload', () => {
			render(FileUpload, {
				props: {
					files: [createMockFile('test.jpg', 1024, 'image/jpeg')],
					uploading: true,
					uploadProgress: { 'test.jpg': 50 }
				}
			});

			expect(screen.getByRole('progressbar')).toBeInTheDocument();
			expect(screen.getByText(/Uploading.*50%/)).toBeInTheDocument();
		});

		it('shows individual file progress overlay', () => {
			render(FileUpload, {
				props: {
					files: [createMockFile('test.jpg', 1024, 'image/jpeg')],
					uploading: true,
					uploadProgress: { 'test.jpg': 75 }
				}
			});

			expect(screen.getByText('75%')).toBeInTheDocument();
		});

		it('shows checkmark when file upload complete', () => {
			const { container } = render(FileUpload, {
				props: {
					files: [createMockFile('test.jpg', 1024, 'image/jpeg')],
					uploading: true,
					uploadProgress: { 'test.jpg': 100 }
				}
			});

			// Complete file has special styling
			const fileItem = container.querySelector('.file-item.complete');
			expect(fileItem).toBeInTheDocument();
		});
	});

	describe('upload action', () => {
		// Skip - needs Svelte 5 event prop migration (createEventDispatcher events don't work with DOM listeners)
		it.skip('dispatches upload event when upload button clicked', async () => {
			const file = createMockFile('test.jpg', 1024, 'image/jpeg');
			const { container } = render(FileUpload, { props: { files: [file] } });

			const uploadHandler = vi.fn();
			const wrapper = container.querySelector('.file-upload');
			wrapper?.addEventListener('upload', uploadHandler as EventListener);

			const uploadBtn = screen.getByRole('button', { name: /Upload 1 file/i });
			await fireEvent.click(uploadBtn);

			expect(uploadHandler).toHaveBeenCalledTimes(1);
		});

		it('disables upload button during upload', () => {
			render(FileUpload, {
				props: {
					files: [createMockFile('test.jpg', 1024, 'image/jpeg')],
					uploading: true
				}
			});

			const uploadBtn = screen.getByRole('button', { name: /Uploading/i });
			expect(uploadBtn).toBeDisabled();
		});

		it('shows correct plural for multiple files', () => {
			render(FileUpload, {
				props: {
					files: [
						createMockFile('1.jpg', 100, 'image/jpeg'),
						createMockFile('2.jpg', 100, 'image/jpeg')
					]
				}
			});

			expect(screen.getByText(/Upload 2 files/i)).toBeInTheDocument();
		});
	});

	describe('file type icons', () => {
		it('shows image preview for image files', () => {
			const { container } = render(FileUpload, {
				props: {
					files: [createMockFile('photo.jpg', 1024, 'image/jpeg')]
				}
			});

			const img = container.querySelector('.file-thumbnail img');
			expect(img).toBeInTheDocument();
			expect(img).toHaveAttribute('src', 'blob:mock-url');
		});

		it('shows video preview with play overlay for video files', () => {
			const { container } = render(FileUpload, {
				props: {
					files: [createMockFile('video.mp4', 1024, 'video/mp4')]
				}
			});

			const video = container.querySelector('.file-thumbnail video');
			expect(video).toBeInTheDocument();
			expect(container.querySelector('.video-overlay')).toBeInTheDocument();
		});

		it('shows file icon for non-media files', () => {
			const { container } = render(FileUpload, {
				props: {
					files: [createMockFile('document.pdf', 1024, 'application/pdf')]
				}
			});

			expect(container.querySelector('.file-icon-pdf')).toBeInTheDocument();
		});
	});

	describe('accessibility', () => {
		it('has proper aria labels', () => {
			render(FileUpload);

			expect(screen.getByRole('region', { name: /file upload area/i })).toBeInTheDocument();
			expect(screen.getByRole('button', { name: /Click or drag files to upload/i })).toBeInTheDocument();
		});

		it('can navigate with keyboard', () => {
			render(FileUpload);
			
			const dropZone = screen.getByRole('button', { name: /Click or drag files to upload/i });
			dropZone.focus();
			
			expect(document.activeElement).toBe(dropZone);
		});

		it('file list has proper role', () => {
			render(FileUpload, {
				props: {
					files: [createMockFile('test.jpg', 1024, 'image/jpeg')]
				}
			});

			expect(screen.getByRole('list', { name: /Selected files/i })).toBeInTheDocument();
			expect(screen.getByRole('listitem')).toBeInTheDocument();
		});
	});

	describe('compact mode', () => {
		it('hides action buttons in compact mode', () => {
			render(FileUpload, {
				props: {
					files: [createMockFile('test.jpg', 1024, 'image/jpeg')],
					compact: true
				}
			});

			expect(screen.queryByText(/Upload 1 file/i)).not.toBeInTheDocument();
			expect(screen.queryByText(/Add More/i)).not.toBeInTheDocument();
		});
	});
});
