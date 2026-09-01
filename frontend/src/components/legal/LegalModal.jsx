import React, { useState, useEffect, useMemo, useRef } from 'react';
import getAppConfig from '../../config/appConfig';
import { getLegalDocuments } from './legalContent';
import './LegalModal.css';

const docIcons = {
  privacy: (
    <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
      <path d="m9 12 2 2 4-4" />
    </svg>
  ),
  terms: (
    <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
      <polyline points="14 2 14 8 20 8" />
      <line x1="16" y1="13" x2="8" y2="13" />
      <line x1="16" y1="17" x2="8" y2="17" />
      <polyline points="10 9 9 9 8 9" />
    </svg>
  ),
  copyright: (
    <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" strokeWidth="2">
      <circle cx="12" cy="12" r="10" />
      <path d="M14.83 14.83a4 4 0 1 1 0-5.66" />
    </svg>
  ),
  contacts: (
    <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z" />
      <polyline points="22,6 12,13 2,6" />
    </svg>
  )
};

function highlightMatches(text, query) {
  if (!query || !query.trim()) return text;
  const parts = text.split(new RegExp(`(${query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi'));
  return parts.map((part, i) =>
    part.toLowerCase() === query.toLowerCase() ? (
      <mark key={i} className="legal-highlight">{part}</mark>
    ) : (
      part
    )
  );
}

const LegalModal = ({ isOpen, onClose, initialDoc = 'privacy' }) => {
  const [activeTab, setActiveTab] = useState(initialDoc);
  const [searchQuery, setSearchQuery] = useState('');
  const contentBodyRef = useRef(null);
  const config = getAppConfig();
  const docs = useMemo(() => getLegalDocuments(config), [config]);

  useEffect(() => {
    if (isOpen) {
      setActiveTab(initialDoc || 'privacy');
      setSearchQuery('');
      const prevOverflow = document.body.style.overflow;
      document.body.style.overflow = 'hidden';
      return () => {
        document.body.style.overflow = prevOverflow;
      };
    }
  }, [isOpen, initialDoc]);

  // Scroll to top of content when tab changes
  useEffect(() => {
    if (contentBodyRef.current) {
      contentBodyRef.current.scrollTop = 0;
    }
  }, [activeTab]);

  useEffect(() => {
    const handleKeyDown = (e) => {
      if (e.key === 'Escape' && isOpen) {
        onClose();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const currentDoc = docs[activeTab] || docs.privacy;

  const filteredSections = searchQuery.trim()
    ? currentDoc.sections.filter(
        (s) =>
          s.heading.toLowerCase().includes(searchQuery.toLowerCase()) ||
          s.content.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : currentDoc.sections;

  const handlePrint = () => {
    window.print();
  };

  return (
    <div className="legal-modal-backdrop" onClick={onClose} role="dialog" aria-modal="true" aria-labelledby="legal-modal-title">
      <div className="legal-modal-dialog" onClick={(e) => e.stopPropagation()}>
        {/* Left Sidebar on desktop / Tab header on mobile */}
        <aside className="legal-modal-sidebar">
          <div className="legal-sidebar-brand">
            <span className="legal-sidebar-badge">Правовой центр</span>
            <h3>Документы сервиса</h3>
            <p className="legal-sidebar-domain">{config.domain}</p>
          </div>

          <nav className="legal-sidebar-nav" aria-label="Юридические документы">
            {Object.values(docs).map((doc) => {
              const isActive = activeTab === doc.id;
              return (
                <button
                  key={doc.id}
                  type="button"
                  className={`legal-sidebar-btn ${isActive ? 'is-active' : ''}`}
                  onClick={() => {
                    setActiveTab(doc.id);
                    setSearchQuery('');
                  }}
                >
                  <span className="legal-sidebar-icon" aria-hidden="true">
                    {docIcons[doc.id] || docIcons.privacy}
                  </span>
                  <div className="legal-sidebar-btn-info">
                    <span className="legal-sidebar-btn-title">{doc.shortTitle}</span>
                    <span className="legal-sidebar-btn-badge">{doc.badge}</span>
                  </div>
                </button>
              );
            })}
          </nav>

          <div className="legal-sidebar-footer">
            <small>По вопросам и обращениям:</small>
            <a href={`mailto:${config.supportEmail}`}>{config.supportEmail}</a>
          </div>
        </aside>

        {/* Right Main Content Pane */}
        <div className="legal-modal-main">
          <header className="legal-main-header">
            <div className="legal-main-title-wrap">
              <span className="legal-main-kicker">{currentDoc.badge}</span>
              <h2 id="legal-modal-title">{currentDoc.title}</h2>
              <span className="legal-main-date">Действующая редакция от {currentDoc.updatedAt}</span>
            </div>

            <div className="legal-main-controls">
              <button
                type="button"
                className="legal-action-btn"
                onClick={handlePrint}
                title="Распечатать или сохранить в PDF"
              >
                <svg viewBox="0 0 24 24" width="16" height="16" stroke="currentColor" fill="none" strokeWidth="2">
                  <path d="M6 9V2h12v7M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2" />
                  <path d="M6 14h12v8H6z" />
                </svg>
                <span>Печать</span>
              </button>

              <button
                type="button"
                className="legal-close-btn"
                onClick={onClose}
                aria-label="Закрыть окно"
                title="Закрыть (Esc)"
              >
                ✕
              </button>
            </div>
          </header>

          {/* Quick Search Toolbar */}
          <div className="legal-search-toolbar">
            <div className="legal-search-input-wrap">
              <svg viewBox="0 0 24 24" width="16" height="16" stroke="currentColor" fill="none" strokeWidth="2" aria-hidden="true">
                <circle cx="11" cy="11" r="8" />
                <path d="m21 21-4.35-4.35" />
              </svg>
              <input
                type="text"
                placeholder="Поиск по тексту выбранного документа..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                aria-label="Поиск по документу"
              />
              {searchQuery && (
                <button
                  type="button"
                  className="legal-search-clear-btn"
                  onClick={() => setSearchQuery('')}
                  aria-label="Очистить поиск"
                >
                  ✕
                </button>
              )}
            </div>
            {searchQuery.trim() && (
              <span className="legal-search-stats">
                Найдено разделов: {filteredSections.length}
              </span>
            )}
          </div>

          {/* Document Content Body */}
          <main className="legal-modal-content-body" ref={contentBodyRef}>
            {filteredSections.length === 0 ? (
              <div className="legal-no-results">
                <p>По запросу «<strong>{searchQuery}</strong>» ничего не найдено в этом документе.</p>
                <button type="button" className="legal-btn-text" onClick={() => setSearchQuery('')}>
                  Сбросить фильтр поиска
                </button>
              </div>
            ) : (
              filteredSections.map((sec, idx) => (
                <section key={idx} className="legal-section-card">
                  <h3 className="legal-section-title">
                    {highlightMatches(sec.heading, searchQuery)}
                  </h3>
                  <div className="legal-section-text">
                    {sec.content.split('\n\n').map((paragraph, pIdx) => (
                      <p key={pIdx}>
                        {paragraph.split('\n').map((line, lIdx) => (
                          <React.Fragment key={lIdx}>
                            {highlightMatches(line, searchQuery)}
                            {lIdx < paragraph.split('\n').length - 1 && <br />}
                          </React.Fragment>
                        ))}
                      </p>
                    ))}
                  </div>
                </section>
              ))
            )}
          </main>

          {/* Modal Footer */}
          <footer className="legal-main-footer">
            <div className="legal-main-footer-meta">
              <span>Домен: <strong>https://{config.domain}</strong></span>
              <span className="legal-meta-sep">•</span>
              <span>Поддержка: <strong>{config.supportEmail}</strong></span>
            </div>
            <button type="button" className="legal-primary-close-btn" onClick={onClose}>
              Ознакомлен(а) и закрыть
            </button>
          </footer>
        </div>
      </div>
    </div>
  );
};

export default LegalModal;
