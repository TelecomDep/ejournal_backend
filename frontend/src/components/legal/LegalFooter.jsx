import React from 'react';
import getAppConfig from '../../config/appConfig';
import './LegalFooter.css';

const LegalFooter = ({ onOpenLegal, className = '', showDisclaimer = false }) => {
  const config = getAppConfig();
  const currentYear = new Date().getFullYear();

  return (
    <footer className={`legal-footer ${className}`}>
      <div className="legal-footer-container">
        <span className="legal-footer-copy">
          © {currentYear} {config.domain}
        </span>

        <div className="legal-footer-links">
          <button
            type="button"
            className="legal-footer-link"
            onClick={() => onOpenLegal && onOpenLegal('privacy')}
          >
            Конфиденциальность (152-ФЗ)
          </button>
          <span className="legal-footer-divider">•</span>
          <button
            type="button"
            className="legal-footer-link"
            onClick={() => onOpenLegal && onOpenLegal('terms')}
          >
            Соглашение
          </button>
          <span className="legal-footer-divider">•</span>
          <button
            type="button"
            className="legal-footer-link"
            onClick={() => onOpenLegal && onOpenLegal('copyright')}
          >
            Авторские права
          </button>
          <span className="legal-footer-divider">•</span>
          <button
            type="button"
            className="legal-footer-link"
            onClick={() => onOpenLegal && onOpenLegal('contacts')}
          >
            Контакты
          </button>
        </div>

        {showDisclaimer && (
          <p className="legal-footer-disclaimer">
            По вопросам удаления данных или авторских прав: <a href={`mailto:${config.supportEmail}`}>{config.supportEmail}</a>
          </p>
        )}
      </div>
    </footer>
  );
};

export default LegalFooter;

